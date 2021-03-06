package file

import (
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/openebs/jiva/types"
)

func New() types.BackendFactory {
	return &Factory{}
}

type Factory struct {
}

type Wrapper struct {
	*os.File
}

func (f *Wrapper) Close() error {
	logrus.Infof("Closing: %s", f.Name())
	return f.File.Close()
}

func (f *Wrapper) Snapshot(name string, userCreated bool, created string) error {
	return nil
}

func (f *Wrapper) Resize(name string, size string) error {
	return nil
}

func (f *Wrapper) Size() (int64, error) {
	stat, err := f.Stat()
	if err != nil {
		return 0, err
	}
	return stat.Size(), nil
}

func (f *Wrapper) SectorSize() (int64, error) {
	return 4096, nil
}

func (f *Wrapper) RemainSnapshots() (int, error) {
	return 1, nil
}

func (f *Wrapper) GetRevisionCounter() (int64, error) {
	return 1, nil
}

func (f *Wrapper) GetVolUsage() (types.VolUsage, error) {
	return types.VolUsage{}, nil
}

func (f *Wrapper) SetRevisionCounter(counter int64) error {
	return nil
}

func (f *Wrapper) SetRebuilding(rebuilding bool) error {
	return nil
}

func (ff *Factory) Create(address string) (types.Backend, error) {
	logrus.Infof("Creating file: %s", address)
	file, err := os.OpenFile(address, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		logrus.Infof("Failed to create file %s: %v", address, err)
		return nil, err
	}

	return &Wrapper{file}, nil
}

func (ff *Factory) SignalToAdd(address string, action string) error {
	return nil
}

func (f *Wrapper) GetMonitorChannel() types.MonitorChannel {
	return nil
}

func (f *Wrapper) GetCloneStatus() (string, error) {
	return "", nil
}

func (f *Wrapper) PingResponse() error {
	return nil
}

func (f *Wrapper) StopMonitoring() {
}

func (f *Wrapper) Sync() (int, error) {
	return 0, nil
}

func (f *Wrapper) Unmap(offset int64, length int64) (int, error) {
	return 0, nil
}
