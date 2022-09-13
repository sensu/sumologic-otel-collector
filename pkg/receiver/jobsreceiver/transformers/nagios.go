package transformers

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/SumoLogic/sumologic-otel-collector/pkg/receiver/jobsreceiver/command"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// NagiosList contains a list of Nagios metrics
type NagiosList []Nagios

// Nagios contains values of Nagios performance data metric
type Nagios struct {
	Label string
	Value float64
	Tags  []NagiosTag
}

type NagiosTag struct {
	Name  string
	Value string
}

// Transform transforms a metric in Nagio perfdata format to OT metrics
func (n NagiosList) Transform() pmetric.Metrics {
	m := pmetric.NewMetrics()

	rm := m.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().UpsertString("host.name", "testHost")

	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName("name")
	sm.Scope().SetVersion("version")

	for _, nagios := range n {
		g := sm.Metrics().AppendEmpty()
		g.SetName(nagios.Label)
		g.SetDataType(pmetric.MetricDataTypeGauge)

		d := g.Gauge().DataPoints().AppendEmpty()
		d.SetStartTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		d.SetDoubleVal(nagios.Value)

		for _, tag := range nagios.Tags {
			d.Attributes().UpsertString(tag.Name, tag.Value)
		}
	}

	return m
}

// ParseNagios parses a Nagios perfdata string into a slice of Nagios struct
func ParseNagios(er *command.ExecutionResponse) NagiosList {
	var nagiosList NagiosList

	// Ensure we have some perfdata metrics and not only human-readable text
	output := strings.Split(er.Output, "|")
	if len(output) != 2 {
		return nagiosList
	}

	// Fetch the perfdata and remove leading & trailing whitespaces
	perfdata := strings.TrimSpace(output[1])

	// Split the perfdata into a slice of metrics
	metrics := strings.Split(perfdata, " ")

	// Create a Nagios metric for each perfdata metrics
	for _, metric := range metrics {
		if metric = strings.TrimSpace(metric); len(metric) == 0 {
			// the token was just whitespace, ignore it
			continue
		}
		// Clear everything after ';' and split the label and the value
		parts := strings.Split(metric, ";")
		parts = strings.Split(parts[0], "=")
		if len(parts) != 2 {
			continue
		}

		// Make sure we don't have any whitespace in our label
		label := strings.Replace(parts[0], " ", "_", -1)

		// Remove all non-numeric characters from the value
		re := regexp.MustCompile(`[^-\d\.]`)
		strValue := re.ReplaceAllString(parts[1], "")

		// Parse the value as a float64
		value, err := strconv.ParseFloat(strValue, 64)
		if err != nil {
			continue
		}

		// Add this metric to our list
		n := Nagios{
			Label: label,
			Value: value,
		}
		nagiosList = append(nagiosList, n)
	}

	return nagiosList
}
