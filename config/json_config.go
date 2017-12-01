package config

import (
	"encoding/json"
	"fmt"
	"io"
)

type jsonStore struct {
	r  io.Reader
	kb map[Key][]byte
}

// NewJSONStore returns a config store backed by a JSON stream.
//
// The first level keys in the JSON stream match the component names and the
// values must be decodeable into the types used to retrieve the config.
func NewJSONStore(r io.Reader) Store {
	return &jsonStore{
		r:  r,
		kb: map[Key][]byte{},
	}
}

func (j *jsonStore) Open() error {
	d := json.NewDecoder(j.r)
	for {
		data := map[Key]*cfgData{}
		if err := d.Decode(&data); err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		// Cache the key and its corresponding json data
		for k, v := range data {
			j.kb[Key(k)] = v.b
		}
	}
}

func (j *jsonStore) Close() {
	// NOOP
}

func (j *jsonStore) Get(config Config) error {
	if config == nil || config.Key().IsNil() {
		// Empty key so just return the default config back
		return nil
	}

	name := config.Key()
	if b, ok := j.kb[name]; ok {
		if e := json.Unmarshal(b, config); e != nil {
			// Bad buffer for the current type but lets keep it around
			// in case the registry is modified with a new type
			// and we can process it in future Get calls
			return e
		}
		return nil
	}
	return fmt.Errorf("%s key not found", name)
}

type cfgData struct {
	b []byte
}

func (d *cfgData) UnmarshalJSON(b []byte) error {
	d.b = b
	return nil
}
