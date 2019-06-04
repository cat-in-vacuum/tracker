package tracker

import (
	"errors"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"io"
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

//
type Track struct {
	Data          MetaData
	Loggable      bool
	callerSkip    int
	messageFormat string
	options       *Options
	Renderer
}

// leverage of options on which be  final rendered the track info
// withErrors - will add an errors field in output if error was send into Update()
// withName - will add a name of calling function
// withSinceStart -  will add duration of since creation instance of Track
// withDuration - will add a duration since previous call Update()
// withTrack - will add a string which  visualize the called function duration
type Options struct {
	withErrors     bool
	withName       bool
	withSinceStart bool
	withDuration   bool
	withTrack      bool
}

func (t *Track) SetMessageFormat(s string) {
	msgFormat = s
}

func New(callerSkip int) *Track {
	t := Track{
		callerSkip: callerSkip,
	}
	t.Data = append(t.Data, Meta{
		start: time.Now(),
		name:  t.trace(),
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
		name:     t.trace(),
		start:    time.Now(),
		dur:      t.Data[len(t.Data)-1].Since(),
		startDif: t.Data[0].Since(),
		err:      err,
	}

	t.Data = append(t.Data, meta)

	if t.Loggable {
		fmt.Println(meta.info())
	}

	return nil
}

// contains meta information about current function
type MetaData []Meta
type Meta struct {
	name     string
	start    time.Time
	dur      time.Duration
	startDif time.Duration
	err      error
}

type RenderOptions struct {
	Divider  int
}

type TableRender struct {
	Out     io.Writer
	Options *RenderOptions
}

type JSONRender struct {
	Options *RenderOptions
}

// returns max duration  of []Track.Data elems
func (m MetaData) MaxDuration() time.Duration {
	var max time.Duration
	for _, v := range m {
		if max < v.dur {
			max = v.dur
		}
	}
	return max
}

func (m MetaData) MinDuration() time.Duration {
	var min time.Duration
	// the first elem will always be with the smallest duration
	// because it create on start, so we skipping it
	for i, e := range m[1:] {
		if i == 0 || e.dur < min {
			min = e.dur
		}
	}
	return min
}

func (iter Meta) Since() time.Duration {
	return time.Since(iter.start)
}

func (iter Meta) info() string {
	return fmt.Sprintf(msgFormat, iter.name, iter.startDif, iter.dur, )
}

func (tbr TableRender) Render(data MetaData, opt *Options) {
	headers := make([]string, 0, len(data))
	headers = createHeaders(headers, opt)
	table := tablewriter.NewWriter(tbr.Out)
	table.SetHeader(headers)
	fmt.Println(headers)

	for i, v := range data {
		var timeLine string

		if v.err == nil {
			v.err = errors.New("")
		}

		step := int(data.MaxDuration()) / tbr.Options.Divider

		for k := 0; k < int(data[i].dur); k += step {
			timeLine = timeLine + "*"
		}

		values := createValues(opt, data[i], timeLine)

		table.Append(values)
	}

	table.Render()
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

func createValues(opt *Options, meta Meta, timeLine string) []string {
	s := make([]string, 0, 5)
	if opt.withName {
		s = append(s, meta.name)
	}
	if opt.withSinceStart {
		s = append(s, meta.startDif.String())
	}
	if opt.withDuration {
		s = append(s, meta.dur.String())
	}
	if opt.withErrors {
		if meta.err == nil {
			s = append(s, "")
		} else {
			s = append(s, meta.err.Error())
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

//returns the name of the function in which it is called
func (t Track) trace() string {
	pc := make([]uintptr, 10)
	runtime.Callers(t.callerSkip, pc)
	f := runtime.FuncForPC(pc[0])
	return f.Name()
}

func (t *Track) Render() {
	t.Renderer.Render(t.Data, t.options)
}
