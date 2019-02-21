package supply_test

import (
	"bytes"
	"github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"netcore_riverbed/supply"
	"os"
)

//go:generate mockgen -source=supply.go --destination=mocks_test.go --package=supply_test

var _ = Describe("Supply", func() {
	var (
		supplier      supply.Supplier
		mockCtrl      *gomock.Controller
		mockManifest  *MockManifest
		mockInstaller *MockInstaller
		mockStager    *MockStager
		mockCommand   *MockCommand
	)

	BeforeEach(func() {
		buffer := new(bytes.Buffer)
		logger := libbuildpack.NewLogger(buffer)

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
		mockStager = NewMockStager(mockCtrl)
		mockCommand = NewMockCommand(mockCtrl)
		mockInstaller = NewMockInstaller(mockCtrl)

		supplier = supply.Supplier{
			Manifest:  mockManifest,
			Installer: mockInstaller,
			Stager:    mockStager,
			Command:   mockCommand,
			Log:       logger,
		}
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Run", func() {

		Context("riverbed appinternals service", func() {
			It(" finds bound appinternals service", func() {
				os.Setenv("VCAP_SERVICES",`{
      "appinternals": [
        {
          "name": "fan_test",
          "instance_name": "fan_test",
          "binding_name": null,
          "credentials": {
            "SB_version": "10.17.1_BL510",
            "DSA_VERSION": "Agent Version: 10.17.1.510 (BL510)"
          },
          "syslog_drain_url": null,
          "volume_mounts": [],
          "label": "appinternals",
          "provider": null,
          "plan": "Riverbed License (Trial or Subscription)",
          "tags": [
            "appinternals"
          ]
        }
      ]
    }`)
				Expect(supplier.IsSupported()).To(Equal(true))
				os.Unsetenv("VCAP_SERVICES")
			})
		})

		Context("riverbed appinternals service", func() {
			It(" unable to find appinternals service", func() {
				Expect(supplier.IsSupported()).To(Equal(false))
			})
		})


		Context("riverbed appinternals service", func() {
			It(" gets URL from credentials served by Service Broker", func() {
				os.Setenv("VCAP_SERVICES",`{
      "appinternals": [
        {
          "name": "fan_test",
          "instance_name": "fan_test",
          "binding_name": null,
          "credentials": {
            "SB_version": "10.17.1_BL510",
            "DSA_VERSION": "Agent Version: 10.17.1.510 (BL510)",
            "DNprofilerUrlLinux": "some_random_url"
          },
          "syslog_drain_url": null,
          "volume_mounts": [],
          "label": "appinternals",
          "provider": null,
          "plan": "Riverbed License (Trial or Subscription)",
          "tags": [
            "appinternals"
          ]
        }
      ]
    }`)

				url, err := supplier.GetDownloadURL()
				Expect(err).To(Succeed())
				Expect(url).To(Equal("some_random_url"))

				os.Unsetenv("VCAP_SERVICES")
			})
		})

		Context("riverbed appinternals service", func() {
			It(" unable to get URL from credentials served by Service Broker", func() {
				url, err := supplier.GetDownloadURL()
				Expect(err).To(Succeed())
				Expect(url).To(Equal(""))

			})
		})

	})
})
