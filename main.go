package main


import (
    "fmt"
	"github.com/jasonlvhit/gocron"
	"os/exec"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"


	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)
  
func set(gauge prometheus.Gauge, value string){
	x,_ := strconv.ParseFloat(value, 64)
	gauge.Set(x)
}

func scrub() {
	_, err := exec.Command("snapraid", "scrub").Output()
	if err != nil {
        log.Fatal(err)
    }
}

func sync() {
	if ( errors ) {
		log.Println("Did not sync pool due to error state")
		return
	}
	_, err := exec.Command("snapraid", "sync").Output()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Synced Pool")
}
	

func status() {
	out, err := exec.Command("snapraid", "status", "-v").Output()
	if err != nil {
        log.Fatal(err)
    }
	out_s := string(out[:])
	out_a := strings.Split(out_s, "\n")
	matches := reNumbers.FindAllString(out_s, -1)
	set(files, matches[1])
	set(hardlinks, matches[2])
	set(symlinks, matches[3])
	set(empty_dirs, matches[4])

	//s := reSnapS.FindAllString(out_s, -1)
	//fmt.Println(s)
	
	x,_ := strconv.ParseFloat(matches[5], 64)
	x = x*1000
	mem_usage.Set(x)

	token  := "none"

	// set sync stataus
	if (strings.Contains(out_s, sync_in_progress_token) ){
		sync_in_progress.Set(1)
	} else {
		sync_in_progress.Set(0)
	} 

	// set error status
	if (strings.Contains(out_s, error_token) ){
		error.Set(1)
		errors=true
	} else  {
		error.Set(0)
		errors=false
	} 
	for _,line := range out_a {
		if (token == "summary") {
			matches := reNumbers.FindAllString(line, -1)
			set(fragmented_files, matches[1])
			set(excess_fragments, matches[2])
			set(used, matches[6])
			token = "none"
		}
		if (strings.Contains(line, zero_sub_token)){
			matches := reNumbers.FindAllString(line, -1)
			if (len(matches) > 0) {
				set(sub_zero, matches[0])
			}
		}
		if (strings.Contains(line, scrub_token)){
			matches := reNumbers.FindAllString(line, -1)
			if (len(matches) > 0) {
				set(scrub_per, matches[0])
			}
		}
		if (strings.Contains(line, scrub_day_token)){
			matches := reNumbers.FindAllString(line, -1)
			if (len(matches) > 2) {
				set(scrub_day_old, matches[0])
				set(scrub_day_med, matches[1])
				set(scrub_day_old, matches[2])
			}
		}
		if (strings.Contains(line, status_end_token)){
			token = "summary"
		}
		if (strings.Contains(line, status_start_token)){
			token="disks"
		}	
	}
	fmt.Println("Updated Status")
}

var (
	reNumbers = regexp.MustCompile(`(\d|\.)+\b`)
	reSnapS = regexp.MustCompile(`\d+\b (MiB|GiB)`)
	errors = false
	status_end_token="--------------------------------------------------------------------------"
	status_start_token="Files  Fragments"
	sync_in_progress_token="You have a sync in progress at "
	error_token="WARNING! The array is NOT fully synced."
	scrub_token="of the array is not scrubbed."
	zero_sub_token="sub-second timestamp"
	scrub_day_token="The oldest block was scrubbed"


	files = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "snapraid_files",
			Help: "The total number of files in array",
	})

	hardlinks = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_hardlinks",
		Help: "The total number of hardlinks in array",
	})

	symlinks = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_symlinks",
		Help: "The total number of symlinks in array",
	})

	empty_dirs = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_emptydirs",
		Help: "The total number of empty dirs in array",
	})

	mem_usage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_array_memory_usage",
		Help: "The total number of empty dirs in array",
	})

	fragmented_files = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_fragmented_files",
		Help: "",
	})
	excess_fragments = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_excess_fragments",
		Help: "",
	})
	used = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_used_percents",
		Help: "",
	})
	sync_in_progress = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_sync_in_progress",
		Help: "",
	})
	error = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_error",
		Help: "",
	})
	sub_zero = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_sub_zero_times",
		Help: "",
	})
	scrub_per = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_not_scrubed_percent",
		Help: "",
	})
	scrub_day_old = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_scrub_days_oldest",
		Help: "",
	})
	scrub_day_med = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_scrub_median",
		Help: "",
	})
	scrub_day_new= promauto.NewGauge(prometheus.GaugeOpts{
		Name: "snapraid_scrub_newest",
		Help: "",
	})
)


func main() {

	gocron.Every(2).Minute().Do(status)
	gocron.Every(12).Hours().Do(sync) 
	gocron.Every(1).Hours().Do(scrub)
	
	gocron.Start()
	
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

}
