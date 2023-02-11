package responses

type PipelineResponse struct {
	Workers               int  `json:"workers"`
	BatchSize             int  `json:"batch_size"`
	BatchDelay            int  `json:"batch_delay"`
	ConfigReloadAutomatic bool `json:"config_reload_automatic"`
	ConfigReloadInterval  int  `json:"config_reload_interval"`
}

type ProcessResponse struct {
	OpenFileDescriptors     int `json:"open_file_descriptors"`
	PeakOpenFileDescriptors int `json:"peak_open_file_descriptors"`
	MaxFileDescriptors      int `json:"max_file_descriptors"`
	Mem                     struct {
		TotalVirtualInBytes int `json:"total_virtual_in_bytes"`
	} `json:"mem"`
	Cpu struct {
		Percent       int `json:"percent"`
		TotalInMillis int `json:"total_in_millis"`
		LoadAverage   struct {
			OneMinute      float64 `json:"1m"`
			FiveMinutes    float64 `json:"5m"`
			FifteenMinutes float64 `json:"15m"`
		} `json:"load_average"`
	} `json:"cpu"`
}

type OsResponse struct {
	Cgroup struct {
		CpuAcct CpuAcctResponse `json:"cpuacct"`
		Cpu     CpuResponse     `json:"cpu"`
	} `json:"cgroup"`
}

type CpuAcctResponse struct {
	ControlGroup string `json:"control_group"`
	UsageNanos   int    `json:"usage_nanos"`
}

type CpuResponse struct {
	ControlGroup    string `json:"control_group"`
	CfsQuotaMicros  int    `json:"cfs_quota_micros"`
	CfsPeriodMicros int    `json:"cfs_period_micros"`
}

type JvmResponse struct {
	Pid               int    `json:"pid"`
	Version           string `json:"version"`
	VMName            string `json:"vm_name"`
	VMVersion         string `json:"vm_version"`
	VMVendor          string `json:"vm_vendor"`
	StartTimeInMillis int64  `json:"start_time_in_millis"`
	Mem               struct {
		HeapInitInBytes    int `json:"heap_init_in_bytes"`
		HeapMaxInBytes     int `json:"heap_max_in_bytes"`
		NonHeapInitInBytes int `json:"non_heap_init_in_bytes"`
		NonHeapMaxInBytes  int `json:"non_heap_max_in_bytes"`
	} `json:"mem"`
	GcCollectors []string `json:"gc_collectors"`
}

type QueueResponse struct {
	EventsCount int `json:"events_count"`
}

type NodeInfoResponse struct {
	Host        string           `json:"host"`
	Version     string           `json:"version"`
	HTTPAddress string           `json:"http_address"`
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Status      string           `json:"status"`
	Snapshot    bool             `json:"snapshot"`
	Pipeline    PipelineResponse `json:"pipeline"`
	Os          OsResponse       `json:"os"`
	Jvm         JvmResponse      `json:"jvm"`
	Queue       QueueResponse    `json:"queue"`
}
