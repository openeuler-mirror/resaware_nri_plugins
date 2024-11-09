package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	timeInterval time.Duration
	connection   string
	duration     string
	threads      string
	timeout      *string
	headers      string
	url          string
	detailInfo   *bool
)

type Queue struct {
	elements []float64
	curSize  int
	maxSize  int
	head     int
	sumVal   float64
}

func NewQueue(maxSize int) *Queue {
	return &Queue{
		elements: make([]float64, maxSize),
		curSize:  0,
		maxSize:  maxSize,
		head:     0,
		sumVal:   float64(0),
	}
}

func (q *Queue) enQueue(element float64) {
	q.sumVal -= q.elements[q.head]
	q.sumVal += element
	q.elements[q.head] = element
	q.head = (q.head + 1) % q.maxSize
	if q.curSize+1 <= q.maxSize {
		q.curSize++
	}
}

func main() {
	flag.DurationVar(&timeInterval, "interval", 5, "time interval for execu wrk")
	flag.StringVar(&connection, "c", "", "connections to keep open")
	flag.StringVar(&duration, "d", "", "duration of test")
	flag.StringVar(&threads, "t", "", "number of threads to use")
	flag.StringVar(&headers, "H", "", "Add header to request")
	timeout = flag.String("timeout", "", "timeout")
	flag.StringVar(&url, "u", "", "test url")
	detailInfo = flag.Bool("detail", false, "detail information")
	flag.Parse()

	ticker := time.NewTicker(time.Second * timeInterval)
	defer ticker.Stop()

	wrkArgs := make([]string, 0)
	if connection != "" {
		wrkArgs = append(wrkArgs, "-c", connection)
	}
	if duration != "" {
		wrkArgs = append(wrkArgs, "-d", duration)
	}
	if threads != "" {
		wrkArgs = append(wrkArgs, "-t", threads)
	}
	wrkArgs = append(wrkArgs, "--latency")
	if *timeout != "" {
		wrkArgs = append(wrkArgs, "--timeout", *timeout)
	}
	if headers != "" {
		wrkArgs = append(wrkArgs, "-H", headers)
	}
	if url == "" {
		fmt.Println("test url cat not be empty")
		os.Exit(-1)
	}
	wrkArgs = append(wrkArgs, url)
	fmt.Printf("wrk args: %s\n\n", strings.Join(wrkArgs, " "))

	q := NewQueue(10)

	testTimes := 0

	for {
		// ./wrk -c 2000 -d 10s -t 20 --latency --timeout 5s -H http://"Connection: close" http://10.211.55.29:30002
		cmd := exec.Command("wrk", wrkArgs...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Execute wrk error: %s\noutput: %s\n", err, output)
			continue
		}
		reqs := phraseOutput(&output)
		val, err := strconv.ParseFloat(reqs, 64)
		if err != nil {
			fmt.Println(err)
		}
		q.enQueue(val)

		fmt.Printf("the %dth test result:\n", testTimes)
		testTimes++

		if *detailInfo {
			fmt.Println(string(output))
		}
		fmt.Printf("Requests/sec: %s, the avg of the last %d times = %f\n", reqs, q.maxSize, q.sumVal/float64(q.curSize))
		fmt.Printf("\n\n")
	}
}

func phraseOutput(output *[]byte) string {
	re := regexp.MustCompile(`Requests/sec:\s+(\d+\.\d+)`)
	matches := re.FindStringSubmatch(string(*output))
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}
