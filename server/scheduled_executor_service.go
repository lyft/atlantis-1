package server

import (
	"context"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/template"
	"time"

	"github.com/google/go-github/v31/github"
	stats "github.com/lyft/gostats"
	"github.com/runatlantis/atlantis/server/events"
	"github.com/runatlantis/atlantis/server/events/metrics"
	"github.com/runatlantis/atlantis/server/events/models"
	"github.com/runatlantis/atlantis/server/logging"
)

type ScheduledExecutorService struct {
	log logging.SimpleLogging

	// jobs
	garbageCollector   JobDefinition
	rateLimitPublisher JobDefinition
}

func NewScheduledExecutorService(
	workingDirIterator events.WorkDirIterator,
	statsScope stats.Scope,
	log logging.SimpleLogging,
	pullCleaner events.PullCleaner,
	githubClient *github.Client,
) *ScheduledExecutorService {

	scheduledScope := statsScope.Scope("scheduled")
	garbageCollector := &GarbageCollector{
		workingDirIterator: workingDirIterator,
		stats:              scheduledScope.Scope("garbagecollector"),
		log:                log,
		cleaner:            pullCleaner,
	}

	garbageCollectorJob := JobDefinition{
		Job: garbageCollector,

		// 5 minutes should probably be the lowest to prevent GH rate limits
		Period: 5 * time.Minute,
	}

	rateLimitPublisher := &RateLimitStatsPublisher{
		client: githubClient,
		stats:  scheduledScope.Scope("ratelimitpublisher"),
		log:    log,
	}

	rateLimitPublisherJob := JobDefinition{
		Job: rateLimitPublisher,

		// since rate limit api doesn't contribute to the rate limit we can call this every minute
		Period: 1 * time.Minute,
	}

	return &ScheduledExecutorService{
		log:                log,
		garbageCollector:   garbageCollectorJob,
		rateLimitPublisher: rateLimitPublisherJob,
	}
}

type JobDefinition struct {
	Job    Job
	Period time.Duration
}

func (s *ScheduledExecutorService) Run() {
	s.log.Info("Scheduled Executor Service started")

	// create tickers
	garbageCollectorTicker := time.NewTicker(s.garbageCollector.Period)
	defer garbageCollectorTicker.Stop()

	rateLimitPublisherTicker := time.NewTicker(s.rateLimitPublisher.Period)
	defer rateLimitPublisherTicker.Stop()

	interrupt := make(chan os.Signal, 1)

	// Stop on SIGINTs and SIGTERMs.
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	for {
		select {
		case <-interrupt:
			s.log.Warn("Received interrupt. Shutting down scheduled executor service")
			return
		case <-garbageCollectorTicker.C:
			go s.garbageCollector.Job.Run()
		case <-rateLimitPublisherTicker.C:
			go s.rateLimitPublisher.Job.Run()
		}
	}
}

type Job interface {
	Run()
}

type RateLimitStatsPublisher struct {
	log    logging.SimpleLogging
	stats  stats.Scope
	client *github.Client
}

func (r *RateLimitStatsPublisher) Run() {

	// since we are calling this at timed intervals it's probably ok to create our own context here.
	// it's not critical to cancel this context
	ctx := context.Background()

	errCounter := r.stats.NewCounter(metrics.ExecutionErrorMetric)
	rateLimitRemainingCounter := r.stats.NewCounter("ratelimitremaining")

	rateLimits, resp, err := r.client.RateLimits(ctx)

	if err != nil {
		r.log.Err("error retrieving rate limits: %s", err)
		errCounter.Inc()
		return
	}

	if resp.StatusCode != 200 {
		r.log.Err("error retrieving rate limits: %s", resp.Status)
		errCounter.Inc()
		return
	}

	rateLimitRemainingCounter.Add(uint64(rateLimits.GetCore().Remaining))
}

var gcStaleClosedPullTemplate = template.Must(template.New("").Parse(
	"Pull Request has been closed for 30 days. Atlantis GC has deleted the locks and plans for the following projects and workspaces:\n" +
		"{{ range . }}\n" +
		"- dir: `{{ .RepoRelDir }}` {{ .Workspaces }}{{ end }}"))

type GCStalePullTemplate struct{}

func (t *GCStalePullTemplate) Execute(wr io.Writer, data interface{}) error {
	return gcStaleClosedPullTemplate.Execute(wr, data)
}

type GarbageCollector struct {
	workingDirIterator events.WorkDirIterator
	stats              stats.Scope
	log                logging.SimpleLogging
	cleaner            events.PullCleaner
}

func (g *GarbageCollector) Run() {
	errCounter := g.stats.NewCounter(metrics.ExecutionErrorMetric)

	pulls, err := g.workingDirIterator.ListCurrentWorkingDirPulls()

	if err != nil {
		g.log.Err("error listing pulls %s", err)
		errCounter.Inc()
	}

	openPullsCounter := g.stats.NewCounter("pulls.open")
	updatedthirtyDaysAgoOpenPullsCounter := g.stats.NewCounter("pulls.open.updated.thirtydaysago")
	closedPullsCounter := g.stats.NewCounter("pulls.closed")
	thirtyDaysAgoClosedPullsCounter := g.stats.NewCounter("pulls.closed.thirtydaysago")
	fiveMinutesAgoClosedPullsCounter := g.stats.NewCounter("pulls.closed.fiveminutesago")

	// we can make this shorter, but this allows us to see trends more clearly
	// to determine if there is an issue or not
	thirtyDaysAgo := time.Now().Add(-720 * time.Hour)
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)

	for _, pull := range pulls {
		logger := g.log.With(fmtLogSrc(pull.BaseRepo, pull.Num)...)

		if pull.State == models.OpenPullState {
			openPullsCounter.Inc()

			if pull.UpdatedAt.Before(thirtyDaysAgo) {
				updatedthirtyDaysAgoOpenPullsCounter.Inc()

				logger.Warn("Pull hasn't been updated for more than 30 days.")
			}
			continue
		}

		// assume only other state is closed
		closedPullsCounter.Inc()

		if pull.ClosedAt.Before(thirtyDaysAgo) {
			thirtyDaysAgoClosedPullsCounter.Inc()

			logger.Warn("Pull closed for more than 30 days but data still on disk")

			err := g.cleaner.CleanUpPull(pull.BaseRepo, pull)

			if err != nil {
				logger.Err("Error cleaning up 30 days old closed pulls %s", err)
				errCounter.Inc()
				return
			}
		}

		// This will allow us to catch leaks as soon as they happen (hopefully)
		if pull.ClosedAt.Before(fiveMinutesAgo) {
			fiveMinutesAgoClosedPullsCounter.Inc()

			logger.Warn("Pull closed for more than 5 minutes but data still on disk")
		}
	}
}

// taken from other parts of the code, would be great to have this in a shared spot
func fmtLogSrc(repo models.Repo, pullNum int) []interface{} {
	return []interface{}{
		"repository", repo.FullName,
		"pull-num", strconv.Itoa(pullNum),
	}
}
