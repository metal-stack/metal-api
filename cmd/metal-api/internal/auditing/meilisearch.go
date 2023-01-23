package auditing

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/meilisearch/meilisearch-go"
	"github.com/robfig/cron"
	"go.uber.org/zap"
)

type meiliAuditing struct {
	client           *meilisearch.Client
	index            *meilisearch.Index
	log              *zap.SugaredLogger
	indexPrefix      string
	rotationInterval Interval
}

func New(c Config) (Auditing, error) {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   c.URL,
		APIKey: c.APIKey,
	})
	v, err := client.GetVersion()
	if err != nil {
		return nil, fmt.Errorf("unable to connect to meilisearch at:%s %w", c.URL, err)
	}
	c.Log.Infow("meilisearch", "connected to", v, "index rotated", c.RotationInterval)

	index := client.Index(c.IndexPrefix)
	if c.RotationInterval != "" {
		index = client.Index(indexName(c.IndexPrefix, c.RotationInterval))
	}

	a := &meiliAuditing{
		client:           client,
		index:            index,
		log:              c.Log.Named("auditing"),
		indexPrefix:      c.IndexPrefix,
		rotationInterval: c.RotationInterval,
	}

	if c.RotationInterval != "" {
		// create a new Index every interval
		cn := cron.New()
		err := cn.AddFunc(string(c.RotationInterval), a.newIndex)
		if err != nil {
			return nil, err
		}
		cn.Start()
	}
	return a, nil
}

func (a *meiliAuditing) Index(keysAndValues ...any) error {
	e := a.toMap(keysAndValues)
	a.log.Debugw("index", "entry", e)
	id := uuid.NewString()
	e["id"] = id
	e["timestamp"] = time.Now()
	e["component"] = "metal-api"
	documents := []map[string]any{e}

	task, err := a.index.AddDocuments(documents, "id")
	if err != nil {
		a.log.Errorw("index", "error", err)
		return err
	}
	stats, _ := a.index.GetStats()
	a.log.Debugw("index", "task status", task.Status, "index stats", stats)
	return nil
}

func (a *meiliAuditing) newIndex() {
	a.log.Debugw("auditing", "create new index", a.rotationInterval)
	a.index = a.client.Index(indexName(a.indexPrefix, a.rotationInterval))
}

func indexName(prefix string, i Interval) string {
	timeFormat := "2006-01-02"

	switch i {
	case HourlyInterval:
		timeFormat = "2006-01-02_15"
	case DailyInterval:
		timeFormat = "2006-01-02"
	case MonthlyInterval:
		timeFormat = "2006-01"
	}

	indexName := prefix + "-" + time.Now().Format(timeFormat)
	return indexName
}

func (a *meiliAuditing) toMap(args []any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	if len(args)%2 != 0 {
		a.log.Errorf("meilisearch pairs of key,value must be provided:%v, not processing", args...)
		return nil
	}
	fields := make(map[string]any)
	for i := 0; i < len(args); {
		key, val := args[i], args[i+1]
		if keyStr, ok := key.(string); ok {
			fields[keyStr] = val
		}
		i += 2
	}
	return fields
}
