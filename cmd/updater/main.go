package main

import (
	"fmt"
	"log"

	"github.com/Houvven/OplusUpdater/pkg/updater"
	"github.com/spf13/cobra"
)

const example = `
  updater CPH2401_11.C.58_0580_202402190800 --region=CN --model=CPH2401
  updater RMX3820_13.1.0.130_0130_202404010000 --region=IN --mode=client_auto --json
  updater A127_13.0_0001 --carrier-id=00000000 --proxy-url=http://localhost:7890
  updater OPD2413_11.A --region=CN --gray
  updater PJX110_11.C --region=CN --mode=taste
`

var rootCmd = &cobra.Command{
	Use:     "updater [OTA_VERSION]",
	Short:   "Query OPlus, OPPO and Realme Mobile OS version updates using official API",
	Example: example,
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		otaVer := args[0]

		mustStr := func(name string) string {
			v, err := cmd.Flags().GetString(name)
			if err != nil {
				log.Fatal(err)
			}
			return v
		}
		mustBool := func(name string) bool {
			v, err := cmd.Flags().GetBool(name)
			if err != nil {
				log.Fatal(err)
			}
			return v
		}

		model := mustStr("model")
		carrier := mustStr("carrier")
		region := mustStr("region")
		guid := mustStr("guid")
		proxy := mustStr("proxy")
		gray := mustBool("gray")
		mode := mustStr("mode")

		if gray && mode == "taste" {
			log.Fatal("reqmode=taste cannot be used together with gray=1")
		}

		result, err := updater.QueryUpdate(&updater.QueryUpdateArgs{
			OtaVersion: otaVer,
			Region:     region,
			Model:      model,
			NvCarrier:  carrier,
			GUID:       guid,
			Proxy:      proxy,
			Gray:       gray,
			Mode:       mode,
		})
		if err != nil {
			log.Fatalf("Error in QueryUpdate: %v", err)
		}
		jsonOut := mustBool("json")
		rawOut := mustBool("raw")
		if rawOut {
			fmt.Println(string(result.DecryptedBodyBytes))
			return
		}
		if jsonOut {
			fmt.Println(string(result.AsJSON()))
			return
		}
		result.PrettyPrint()
	},
}

func init() {
	rootCmd.Flags().StringP("model", "m", "", "Device model, e.g., RMX3820;")
	rootCmd.Flags().StringP("region", "r", "CN", "Server region: CN (default), EU, IN, SG, RU, TR, TH or GL")
	rootCmd.Flags().StringP("carrier", "c", "", "Carrier ID, empty to use region default")
	rootCmd.Flags().StringP("guid", "g", "", "GUID, empty to use zeros")
	rootCmd.Flags().StringP("proxy", "p", "", "Proxy URL, e.g., type://user:password@host:port")
	rootCmd.Flags().Bool("gray", false, "Use gray update server (CN only)")
	rootCmd.Flags().String("mode", "manual", "Request mode: manual, server_auto, client_auto or taste; taste is incompatible with --gray")
	rootCmd.Flags().Bool("json", false, "Output JSON without color")
	rootCmd.Flags().Bool("raw", false, "Output raw decrypted payload string")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
