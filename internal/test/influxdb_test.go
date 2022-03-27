package test

import (
	"context"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"strconv"
	"testing"
	"time"
)

var (
	token = "2kEgsO87u7nv80byZBa0yh6mDe_80zKh3CZ8zw1c2VkF64OL0-SN1trt-IanQTVVzkcCx2JTJnGKnFO63HjAnQ=="
)

func TestInfluxDB_Write(t *testing.T) {
	client := influxdb2.NewClient("http://localhost:8086", token)
	ping, err := client.Ping(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !ping {
		t.Fatal("failed to connect influxdb")
	}
	defer client.Close()

	username := "test"
	writeAPI := client.WriteAPI("test", username)
	defer writeAPI.Flush()

	tags := map[string]string{
		"deviceClassID": "0",
	}
	fields := map[string]float64{
		"current": 0,
		"voltage": 0,
	}

	now := time.Now().UTC()
	for i := 0; i < 100; i++ {
		tags["deviceClassID"] = fmt.Sprintf("%d", i%2)
		fields["current"] = float64(i)
		fields["voltage"] = float64(i * 2)

		measurement := strconv.Itoa(i % 4)
		point := write.NewPointWithMeasurement(measurement)
		point.SetTime(now)
		for k, v := range tags {
			point.AddTag(k, v)
		}
		for k, v := range fields {
			point.AddField(k, v)
		}
		point.SortFields().SortTags()
		writeAPI.WritePoint(point)
		now = now.Add(time.Second)
	}
}
func TestInfluxdb_Query(t *testing.T) {
	client := influxdb2.NewClient("http://localhost:8086", token)
	ping, err := client.Ping(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !ping {
		t.Fatal("failed to connect influxdb")
	}
	defer client.Close()

	queryApi := client.QueryAPI("test")
	result, err := queryApi.Query(context.Background(),
		`from(bucket: "test")
  				|> range(start: -1h)
  				|> filter(fn: (r) => r["_measurement"] == "test-0")
				|> filter(fn: (r) => r["_field"] == "voltage")`,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer result.Close()

	for result.Next() {
		t.Log(result.Record().Value())
	}
}
func TestInfluxdb_Delete(t *testing.T) {
	client := influxdb2.NewClient("http://localhost:8086", token)
	ping, err := client.Ping(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !ping {
		t.Fatal("failed to connect influxdb")
	}
	defer client.Close()

	deleteAPI := client.DeleteAPI()
	for _, m := range []string{"1", "0"} {
		name := fmt.Sprintf(`deviceClassID="%s"`, m)
		err = deleteAPI.DeleteWithName(
			context.Background(),
			"test",
			"test",
			time.Now().UTC().Add(-30*time.Minute),
			time.Now().UTC(),
			name,
		)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestInfluxdb_Task(t *testing.T) {
	client := influxdb2.NewClient("http://localhost:8086", token)
	ping, err := client.Ping(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !ping {
		t.Fatal("failed to connect influxdb")
	}
	defer client.Close()

	tasksAPI := client.TasksAPI()
	flux := `// Defines a data source
			data = from(bucket: "test")
				|> range(start: -task.every)
				|> filter(fn: (r) => r.deviceClassID == "0" and r._field == "current")
			
			data
				|> aggregateWindow(fn: mean, every: task.every, createEmpty: false)
				// Stores the aggregated data in a new bucket
				|> to(bucket: "warning_detect", org: "test")`
	status := domain.TaskStatusTypeActive
	every := "1s"
	task, err := tasksAPI.CreateTask(context.Background(), &domain.Task{
		Every:  &every,
		Flux:   flux,
		Name:   "test_task",
		OrgID:  "883d240e28ee6fae",
		Status: &status,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = tasksAPI.RunManually(context.Background(), task)
	if err != nil {
		t.Fatal(err)
	}
}
