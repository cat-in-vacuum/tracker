package tracker

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io"
	"log"
	"runtime"
	"time"
)

var (
	msgFormat = "function:[%s]|sinceStart:[%s]|duration:[%s]|"
)

// Renderer track trace must implement Render() , but should not be aware of the output.
// In this package are implemented two Renderer`s  - table render and json render,
// but you can use other.
type Renderer interface {
	Render(metadata MetaData, opt *Options)
}

// the track is
type Track struct {
	Data          MetaData `json:"trackedData,omitempty"`
	Loggable      bool
	callerSkip    int
	messageFormat string
	options       *Options
	Renderer
}

// contains meta information about current function
type MetaData []Meta
type Meta struct {
	Name     string        `json:"name"`
	Start    time.Time     `json:"start"`
	Dur      time.Duration `json:"dur"`
	StartDif time.Duration `json:"start_dif"`
	Err      error         `json:"error"`
}

// leverage of options for build info
// withErrors - will add an errors fields in output if error sent into Update()
// withName - will add a name of calling function
// withSinceStart -  will add duration of since creation instance of Track
// withDuration - will add a duration since previous call Update()
// withTrack - will add a string which  visualize the called function duration
type Options struct {
	withErrors,
	withName,
	withSinceStart,
	withDuration,
	withTrack,
	withLink bool
}

func (t *Track) SetMessageFormat(s string) {
	msgFormat = s
}

func New(callerSkip int) *Track {
	t := Track{
		callerSkip: callerSkip,
	}
	t.Data = append(t.Data, Meta{
		Start: time.Now(),
		Name:  trace(t.callerSkip),
	})

	if t.Loggable {
		fmt.Println(t.Data[0].info())
	}
	return &t
}

// Track.Update() append elem into t.Data which contain the invoke time ,
// duration since of previous invoke, name of function who call Update()
func (t *Track) Update(err error) error {
	if len(t.Data) < 1 {
		return errors.New("at first need to invoke New(int)")
	}

	meta := Meta{
		Name:     trace(t.callerSkip),
		Start:    time.Now(),
		Dur:      t.Data[len(t.Data)-1].Since(),
		StartDif: t.Data[0].Since(),
		Err:      err,
	}

	t.Data = append(t.Data, meta)

	if t.Loggable {
		fmt.Println(meta.info())
	}

	return nil
}

type RenderOptions struct {
	Divider int
}

type TableRender struct {
	Out     io.Writer
	Options *RenderOptions
}

type JSONRender struct {
	Out     io.Writer
	Options *RenderOptions
}

// returns max duration  of []Track.Data elems
func (m MetaData) MaxDuration() time.Duration {
	var max time.Duration
	for _, v := range m {
		if max < v.Dur {
			max = v.Dur
		}
	}
	return max
}

func (m MetaData) MinDuration() time.Duration {
	var min time.Duration
	// the first elem will always be with the smallest duration
	// because it create on start, skip it
	for i, e := range m[1:] {
		if i == 0 || e.Dur < min {
			min = e.Dur
		}
	}
	return min
}

func (iter Meta) Since() time.Duration {
	return time.Since(iter.Start)
}

func (iter Meta) info() string {
	return fmt.Sprintf(msgFormat, iter.Name, iter.StartDif, iter.Dur, )
}

func (tbr TableRender) Render(data MetaData, opt *Options) {
	headers := make([]string, 0, len(data))
	headers = createHeaders(headers, opt)
	table := tablewriter.NewWriter(tbr.Out)
	table.SetHeader(headers)

	for i, v := range data {
		var timeLine string

		if v.Err == nil {
			v.Err = errors.New("")
		}

		step := int(data.MaxDuration()) / tbr.Options.Divider

		for k := 0; k < int(data[i].Dur); k += step {
			timeLine = timeLine + "*"
		}

		row := createRow(opt, data[i], timeLine)

		table.Append(row)
	}

	table.Render()
}

func (jsr JSONRender) Render(data MetaData, opt *Options) {
	payload, err := json.MarshalIndent(data, "", "	")
	if err != nil {
		log.Printf("err:%s; error marshaling data", err.Error(), )
	}
	_, err = jsr.Out.Write(payload)
	if err != nil {
		log.Printf("err:%s; error marshaling data", err.Error(), )
	}
}

func createHeaders(s []string, opt *Options) []string {
	if opt.withName {
		s = append(s, "func.name")
	}
	if opt.withSinceStart {
		s = append(s, "since.start")
	}
	if opt.withDuration {
		s = append(s, "duration")
	}
	if opt.withErrors {
		s = append(s, "errors")

	}
	if opt.withTrack {
		s = append(s, "track")
	}
	return s
}

func createRow(opt *Options, meta Meta, timeLine string) []string {
	s := make([]string, 0, 5)
	if opt.withName {
		s = append(s, meta.Name)
	}
	if opt.withSinceStart {
		s = append(s, meta.StartDif.String())
	}
	if opt.withDuration {
		s = append(s, meta.Dur.String())
	}
	if opt.withErrors {
		if meta.Err == nil {
			s = append(s, "")
		} else {
			s = append(s, meta.Err.Error())
		}
	}
	if opt.withTrack {
		s = append(s, timeLine)
	}
	return s
}

func (t *Track) Configure() *Options {
	t.options = new(Options)
	return t.options
}

func (o *Options) WithErrors() *Options {
	o.withErrors = true
	return o
}

func (o *Options) WithName() *Options {
	o.withName = true
	return o
}

func (o *Options) WithSinceStart() *Options {
	o.withSinceStart = true
	return o
}

func (o *Options) WithDuration() *Options {
	o.withDuration = true
	return o
}

func (o *Options) WithTrack() *Options {
	o.withTrack = true
	return o
}

func (t *Track) SetRenderer(render Renderer) {
	t.Renderer = render
}

func (t *Track) Render() {
	t.Renderer.Render(t.Data, t.options)
}

//returns the name of the function in which it is called
func trace(skip int) string {
	pc := make([]uintptr, skip)
	runtime.Callers(skip, pc)
	f := runtime.FuncForPC(pc[0])
	return f.Name()
}
