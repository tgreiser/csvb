package csvb

import (
	"encoding/csv"
	"errors"
	"github.com/oleiade/reflections"
	"io"
	"reflect"
	"strconv"
	"time"
)

var (
	ErrNoCustomHeader = errors.New("missing custom header metadata")
	ErrNoHeader       = errors.New("missing header metadata")
)

type Options struct {
	Separator  rune
	NullMarker string
	TimeZone   *time.Location
	Header     map[int]string
}

type Binder struct {
	csv  *csv.Reader
	meta map[int]string
	opts *Options
}

type Row struct {
	data map[string]string
	opts *Options
}

func NewBinder(reader io.Reader, opts *Options) (*Binder, error) {

	csv := csv.NewReader(reader)

	if opts == nil {
		opts = &Options{}
	} else {
		if opts.Separator == 0 {
			opts.Separator = ','
		}
		csv.Comma = opts.Separator
	}

	if opts.TimeZone == nil {
		opts.TimeZone = time.UTC
	}

	var meta map[int]string

	if len(opts.Header) == 0 {
		header, err := csv.Read()

		if err != nil {
			return nil, err
		}

		meta = make(map[int]string)
		for i, col := range header {
			meta[i] = col
		}
		if len(meta) == 0 {
			return nil, ErrNoHeader
		}
	} else {
		meta = opts.Header
		if len(meta) == 0 {
			return nil, ErrNoCustomHeader
		}
	}

	return &Binder{csv: csv, meta: meta, opts: opts}, nil
}

func (b *Binder) ReadRow() (Row, error) {
	row, err := b.csv.Read()
	if err != nil {
		return Row{}, err
	}
	data := make(map[string]string)
	for i, v := range row {
		if len(v) > 0 && v != b.opts.NullMarker {
			k := b.meta[i]
			data[k] = v
		}
	}
	return Row{data: data, opts: b.opts}, nil
}

func (b *Binder) ForEach(f func(Row) (bool, error)) error {

	for {
		row, err := b.ReadRow()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		hasNext, err := f(row)
		if err != nil {
			return err
		}
		if !hasNext {
			break
		}
	}

	return nil
}

func (r *Row) Bind(x interface{}, strategy map[string]string) error {

	for src, dest := range strategy {

		data, ok := r.data[src]

		if ok {
			k, err := reflections.GetFieldKind(x, dest)
			if err != nil {
				return err
			}

			switch k {
			case reflect.String:
				{
					reflections.SetField(x, dest, data)
				}
			case reflect.Int64:
				{
					i, err := strconv.ParseInt(data, 10, 64)
					if err != nil {
						return err
					}
					reflections.SetField(x, dest, i)
				}
			case reflect.Struct:
				{
					value, err := reflections.GetField(x, dest)
					if err != nil {
						return err
					}
					if reflect.TypeOf(value) == reflect.TypeOf(time.Now()) {
						date, err := time.ParseInLocation("2006-01-02 15:04:05", data, r.opts.TimeZone)
						if err != nil {
							return err
						}
						reflections.SetField(x, dest, date)
					}
				}
			}
		}
	}

	return nil
}
