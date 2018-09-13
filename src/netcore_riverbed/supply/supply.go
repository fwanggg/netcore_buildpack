package supply

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Stager interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/stager.go
	BuildDir() string
	DepDir() string
	DepsIdx() string
	DepsDir() string
}

type Manifest interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/manifest.go
	AllDependencyVersions(string) []string
	DefaultVersion(string) (libbuildpack.Dependency, error)
	RootDir() string
}

type Installer interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/installer.go
	InstallDependency(libbuildpack.Dependency, string) error
	InstallOnlyVersion(string, string) error
}

type Command interface {
	//TODO: See more options at https://github.com/cloudfoundry/libbuildpack/blob/master/command.go
	Execute(string, io.Writer, io.Writer, string, ...string) error
	Output(dir string, program string, args ...string) (string, error)
}

type Supplier struct {
	Manifest  Manifest
	Installer Installer
	Stager    Stager
	Command   Command
	Log       *libbuildpack.Logger
}

func (s *Supplier) Run() error {
	s.Log.BeginStep("Supplying netcore_riverbed")

	// TODO: Install any dependencies here...
	if err:=os.MkdirAll(filepath.Join(s.Stager.DepDir(),"profile.d"),0777); err!=nil{
		return fmt.Errorf("os.MkdirAll: %s", err)
	}

	//s.Stager.DepDir() is depdir in staging process
	//%DEPS_DIR% should be used at runtime
	if err:=ioutil.WriteFile(filepath.Join(s.Stager.DepDir(),"profile.d","riverbed.sh"),[]byte(`
	export CORECLR_PROFILER={cf0d821e-299b-5307-a3d8-b283c03916dd}
	export CORECLR_ENABLE_PROFILING=1
	export CORECLR_PROFILER_PATH=$DEPS_DIR/`+ s.Stager.DepsIdx() +`/CorProfiler.so
	`), 0777) ; err!=nil{
		return fmt.Errorf("ioutil.WRiteFIle: %s", err)
	}

	if err:=libbuildpack.CopyFile(filepath.Join(s.Manifest.RootDir(),"CorProfiler.so"),filepath.Join(s.Stager.DepDir(),"CorProfiler.so")); err!=nil{
		return fmt.Errorf("mv coreprofiler.so: %s", err)
	}

	//fmt.Println(exec.Command("find", s.Stager.DepDir()).Output())
	return nil
}
