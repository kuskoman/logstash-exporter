package responses

import "time"

type PipelineResponse struct {
	Workers    int `json:"workers"`
	BatchSize  int `json:"batch_size"`
	BatchDelay int `json:"batch_delay"`
}

type PoolResponse struct {
	PeakMaxInBytes   int64 `json:"peak_max_in_bytes"`
	MaxInBytes       int64 `json:"max_in_bytes"`
	CommittedInBytes int64 `json:"committed_in_bytes"`
	PeakUsedInBytes  int64 `json:"peak_used_in_bytes"`
	UsedInBytes      int64 `json:"used_in_bytes"`
}

type CollectorResponse struct {
	CollectionCount        int `json:"collection_count"`
	CollectionTimeInMillis int `json:"collection_time_in_millis"`
}

type JvmResponse struct {
	Threads struct {
		Count     int `json:"count"`
		PeakCount int `json:"peak_count"`
	} `json:"threads"`
	Mem struct {
		HeapUsedPercent         int   `json:"heap_used_percent"`
		HeapCommittedInBytes    int64 `json:"heap_committed_in_bytes"`
		HeapMaxInBytes          int64 `json:"heap_max_in_bytes"`
		HeapUsedInBytes         int64 `json:"heap_used_in_bytes"`
		NonHeapUsedInBytes      int64 `json:"non_heap_used_in_bytes"`
		NonHeapCommittedInBytes int64 `json:"non_heap_committed_in_bytes"`
		Pools                   struct {
			Young    PoolResponse `json:"young"`
			Old      PoolResponse `json:"old"`
			Survivor PoolResponse `json:"survivor"`
		} `json:"pools"`
	} `json:"mem"`
	Gc struct {
		Collectors struct {
			Young CollectorResponse `json:"young"`
			Old   CollectorResponse `json:"old"`
		} `json:"collectors"`
	} `json:"gc"`
	UptimeInMillis int `json:"uptime_in_millis"`
}

type ProcessResponse struct {
	OpenFileDescriptors     int64 `json:"open_file_descriptors"`
	PeakOpenFileDescriptors int64 `json:"peak_open_file_descriptors"`
	MaxFileDescriptors      int64 `json:"max_file_descriptors"`
	Mem                     struct {
		TotalVirtualInBytes int64 `json:"total_virtual_in_bytes"`
	} `json:"mem"`
	CPU struct {
		TotalInMillis int64 `json:"total_in_millis"`
		Percent       int   `json:"percent"`
		LoadAverage   struct {
			OneM     float64 `json:"1m"`
			FiveM    float64 `json:"5m"`
			FifteenM float64 `json:"15m"`
		} `json:"load_average"`
	} `json:"cpu"`
}

type EventsResponse struct {
	In                        int64 `json:"in"`
	Filtered                  int64 `json:"filtered"`
	Out                       int64 `json:"out"`
	DurationInMillis          int64 `json:"duration_in_millis"`
	QueuePushDurationInMillis int64 `json:"queue_push_duration_in_millis"`
}

type FlowResponse struct {
	InputThroughput struct {
		Current  float64 `json:"current"`
		Lifetime float64 `json:"lifetime"`
	} `json:"input_throughput"`
	FilterThroughput struct {
		Current  float64 `json:"current"`
		Lifetime float64 `json:"lifetime"`
	} `json:"filter_throughput"`
	OutputThroughput struct {
		Current  float64 `json:"current"`
		Lifetime float64 `json:"lifetime"`
	} `json:"output_throughput"`
	QueueBackpressure struct {
		Current  float64 `json:"current"`
		Lifetime float64 `json:"lifetime"`
	} `json:"queue_backpressure"`
	WorkerConcurrency struct {
		Current  float64 `json:"current"`
		Lifetime float64 `json:"lifetime"`
	} `json:"worker_concurrency"`
}

type SinglePipelineResponse struct {
	Monitoring PipelineLogstashMonitoringResponse `json:".monitoring-logstash"`
	Events     EventsResponse                     `json:"events"`
	Flow       FlowResponse                       `json:"flow"`
	Plugins    struct {
		Inputs []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Events struct {
				Out                       int `json:"out"`
				QueuePushDurationInMillis int `json:"queue_push_duration_in_millis"`
			} `json:"events"`
		} `json:"inputs"`
		Codecs []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Decode struct {
				Out              int `json:"out"`
				WritesIn         int `json:"writes_in"`
				DurationInMillis int `json:"duration_in_millis"`
			} `json:"decode"`
			Encode struct {
				WritesIn         int `json:"writes_in"`
				DurationInMillis int `json:"duration_in_millis"`
			} `json:"encode"`
		} `json:"codecs"`
		Filters []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Events struct {
				Out              int `json:"out"`
				In               int `json:"in"`
				DurationInMillis int `json:"duration_in_millis"`
			} `json:"events"`
		} `json:"filters"`
		Outputs []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Events struct {
				Out              int `json:"out"`
				In               int `json:"in"`
				DurationInMillis int `json:"duration_in_millis"`
			} `json:"events"`
			Documents struct {
				Successes            int `json:"successes"`
				NonRetryableFailures int `json:"non_retryable_failures"`
			} `json:"documents"`
			BulkRequests struct {
				WithErrors int            `json:"with_errors"`
				Responses  map[string]int `json:"responses"`
			} `json:"bulk_requests"`
		} `json:"outputs"`
	} `json:"plugins"`
	Reloads PipelineReloadResponse `json:"reloads"`
	Queue   struct {
		Type                string `json:"type"`
		EventsCount         int64  `json:"events_count"`
		QueueSizeInBytes    int64  `json:"queue_size_in_bytes"`
		MaxQueueSizeInBytes int64  `json:"max_queue_size_in_bytes"`
	} `json:"queue"`
	DeadLetterQueue struct {
		MaxQueueSizeInBytes int `json:"max_queue_size_in_bytes"`
		// todo: research how LastError is returned
		QueueSizeInBytes int64  `json:"queue_size_in_bytes"`
		DroppedEvents    int64  `json:"dropped_events"`
		ExpiredEvents    int64  `json:"expired_events"`
		StoragePolicy    string `json:"storage_policy"`
	} `json:"dead_letter_queue"`
	Hash        string `json:"hash"`
	EphemeralID string `json:"ephemeral_id"`
}

type PipelineLogstashMonitoringResponse struct {
	Events struct {
		Out                       int `json:"out"`
		Filtered                  int `json:"filtered"`
		In                        int `json:"in"`
		DurationInMillis          int `json:"duration_in_millis"`
		QueuePushDurationInMillis int `json:"queue_push_duration_in_millis"`
	} `json:"events"`
	Flow    FlowResponse `json:"flow"`
	Plugins struct {
		Inputs  []interface{} `json:"inputs"`
		Codecs  []interface{} `json:"codecs"`
		Filters []interface{} `json:"filters"`
		Outputs []interface{} `json:"outputs"`
	} `json:"plugins"`
	Reloads PipelineReloadResponse `json:"reloads"`
	Queue   interface{}            `json:"queue,omitempty"`
}

type PipelineReloadResponse struct {
	LastFailureTimestamp *time.Time `json:"last_failure_timestamp,omitempty"`
	Successes            int        `json:"successes"`
	Failures             int        `json:"failures"`
	LastSuccessTimestamp *time.Time `json:"last_success_timestamp,omitempty"`
	LastError            LastError  `json:"last_error,omitempty"`
}

type LastError struct {
	Message   string   `json:"message"`
	Backtrace []string `json:"backtrace"`
}

type ReloadResponse struct {
	Successes int `json:"successes"`
	Failures  int `json:"failures"`
}

type OsResponse struct {
	Cgroup struct {
		Cpu struct {
			CfsPeriodMicros int64 `json:"cfs_period_micros"`
			CfsQuotaMicros  int64 `json:"cfs_quota_micros"`
			Stat            struct {
				TimeThrottledNanos     int64 `json:"time_throttled_nanos"`
				NumberOfTimesThrottled int64 `json:"number_of_times_throttled"`
				NumberOfElapsedPeriods int64 `json:"number_of_elapsed_periods"`
			} `json:"stat"`
			ControlGroup string `json:"control_group"`
		} `json:"cpu"`
		Cpuacct struct {
			UsageNanos   int64  `json:"usage_nanos"`
			ControlGroup string `json:"control_group"`
		} `json:"cpuacct"`
	} `json:"cgroup"`
}

type QueueResponse struct {
	EventsCount int `json:"events_count"`
}

// NodeStatsResponse is the response from the _node/stats API.
type NodeStatsResponse struct {
	Host        string           `json:"host"`
	Version     string           `json:"version"`
	HttpAddress string           `json:"http_address"`
	Id          string           `json:"id"`
	Name        string           `json:"name"`
	EphemeralId string           `json:"ephemeral_id"`
	Status      string           `json:"status"`
	Snapshot    bool             `json:"snapshot"`
	Pipeline    PipelineResponse `json:"pipeline"`
	Jvm         JvmResponse      `json:"jvm"`
	Process     ProcessResponse  `json:"process"`
	Events      EventsResponse   `json:"events"`
	Flow        FlowResponse     `json:"flow"`
	Reloads     ReloadResponse   `json:"reloads"`
	Os          OsResponse       `json:"os"`
	Queue       QueueResponse    `json:"queue"`

	Pipelines map[string]SinglePipelineResponse `json:"pipelines"`
}
