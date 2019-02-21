package supply

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
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

const APPINTERNALS_FOLDER string = "riverbed_appinternals_agent"

const ZIPFILE string = "dn.profiler.linux.zip"

/*
step 1 : check if riverbed service is created and bound to this app (a tag in credentials)

step 2:  download the netcore profiler zip file from service broker.

step 3:  setup the environment variables

step 4:  set JBP_VERSION just like what JBP does (change name though)

*/

func (s *Supplier) Run() error {

	s.Log.BeginStep("Supplying Riverbed AppInternals Buildpack for .NET Core")

	//step 1 : check if riverbed service is created and bound to this app (a tag in credentials)
	s.Log.BeginStep("Checking if the app is bound to AppInternals service...")

	supported, err := s.IsSupported()

	if err != nil{
		return fmt.Errorf("Failed to check if App is bound to AppInternals service: %s ", err)
	}

	if !supported {
		fmt.Printf("The app is not bound to AppInternals service")
		return nil
	}

	// step 2:  download the netcore profiler zip file from service broker.

	s.Log.BeginStep("Downloading .NET Core artifacts from Service Broker...")


	url, err:= s.GetDownloadURL()

	if err != nil{
		return fmt.Errorf("Failed to Get the download URL from Credentials")
	}

	if len(url) == 0 {
		fmt.Printf("DNprofilerUrlLinux is not found in credentials")
		//do we need any backup approach?
		return nil;
	}

	s.Log.BeginStep("Download to DepDir: " + s.Stager.DepDir())

	if err:= s.DownloadFile(filepath.Join(s.Stager.DepDir(), ZIPFILE), url); err != nil{
		return fmt.Errorf("Failed to download %s from %s", ZIPFILE, url)
	}

	//create "riverbed_appinternals_agent" directory as root dir for our agent artifacts
	if err:=os.MkdirAll(s.AgentDir(),0777); err != nil {
		return fmt.Errorf("creating %s: %s", s.AgentDir(), err)
	}

	if err:=libbuildpack.ExtractZip(filepath.Join(s.Stager.DepDir(), ZIPFILE),s.AgentDir()); err!=nil{
		return fmt.Errorf("extarct zip: %s", err)
	}

	if err:=os.Remove(filepath.Join(s.Stager.DepDir(), ZIPFILE)); err!=nil{
		return fmt.Errorf("remove zip: %s", err)
	}

	if err:=os.MkdirAll(filepath.Join(s.Stager.DepDir(),"profile.d"),0777); err!=nil{
		return fmt.Errorf("os.MkdirAll: %s", err)
	}

	//step 3:  setup the environment variables
	//note: no need to set DSA_HOST, the profiler will check for CF_INSTANCE_IP on its own and use that as DSA_HOST
	s.Log.BeginStep("Setting Environment Variables for Instrumentation...")

	phome := "$DEPS_DIR/"+ s.Stager.DepsIdx() +"/agent/"
	s.Log.BeginStep("Using " + phome + " as Panorama Home Directory")

	if err:=ioutil.WriteFile(filepath.Join(s.Stager.DepDir(),"profile.d","riverbed.sh"),[]byte(`
	export CORECLR_PROFILER={cf0d821e-299b-5307-a3d8-b283c03916dd}
	export CORECLR_ENABLE_PROFILING=1
	export CORECLR_PROFILER_PATH=` + phome + `/lib/libAwDotNetProf64.so
    export DOTNET_SHARED_STORE=` + phome + `/install/dotnet/store
    export DOTNET_ADDITIONAL_DEPS=` + phome + `/install/dotnet/additionalDeps/Riverbed.AppInternals.DotNetCore
    export AIX_INSTRUMENT_ALL=1
    export RVBD_IN_PCF=1
    export RVBD_AGENT_FILES=1
	`), 0777) ; err!=nil{
		return fmt.Errorf("ioutil.WRiteFIle: %s", err)
	}

	//step 4:  set JBP_VERSION just like what JBP does (change name though)

	return nil
}

func (s *Supplier) IsSupported() (bool, error){

	//same as map[String]interface{}, just preference
	//Empty interfaces are used by code that handles values of unknown type.
	m := make(map[string]interface{})
	data := os.Getenv("VCAP_SERVICES")

	if len(data) == 0{
		fmt.Printf("The app is not bound to any service.")
		return false, nil
	}

	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return false, err
	}

	//{
	//      "appinternals": [
	//        {
	//          "name": "fan_test",
	//          "instance_name": "fan_test",
	//          "binding_name": null,
	//          "credentials": {
	//            "SB_version": "10.17.1_BL510",
	//            "DSA_VERSION": "Agent Version: 10.17.1.510 (BL510)"
	//          },
	//          "syslog_drain_url": null,
	//          "volume_mounts": [],
	//          "label": "appinternals",
	//          "provider": null,
	//          "plan": "Riverbed License (Trial or Subscription)",
	//          "tags": [
	//            "appinternals"
	//          ]
	//        }
	//      ]
	//    }

	for _, val := range m{
		for _, sa := range val.([]interface{}){
			for aixKey, aixVal := range sa.(map[string]interface{}){
				if aixKey == "tags"{
					for _, tag:= range aixVal.([]interface{}){
						t, ok := tag.(string)
						if ok && strings.Contains(strings.ToLower(t), "appinternals"){
							return true, nil
						}
					}
				}
			}
		}


	}

	return false, nil
}

func (s *Supplier) GetDownloadURL() (string, error){

	//same as map[String]interface{}, just preference
	//Empty interfaces are used by code that handles values of unknown type.

	data := os.Getenv("VCAP_SERVICES")
	if len(data) == 0 {
		fmt.Printf("VCAP_SERVICES is empty.")
		return "", nil
	}

	m := make(map[string]interface{})

	if err := json.Unmarshal([]byte(data), &m); err != nil {
		return "", err
	}


	for _, val := range m{
		for _, sa := range val.([]interface{}){
			for aixKey, aixVal := range sa.(map[string]interface{}){
				if aixKey == "credentials"{
					for k, v:= range aixVal.(map[string]interface{}){
						//DNprofilerUrlLinux
						if strings.Contains(strings.ToLower(k), "dnprofilerurllinux"){
							return v.(string), nil
						}
					}
				}
			}
		}


	}

	return "", nil
}

func (s *Supplier) DownloadFile(filepath string, url string) error {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func (s *Supplier) AgentDir() string{
	return filepath.Join(s.Stager.DepDir(), APPINTERNALS_FOLDER)
}