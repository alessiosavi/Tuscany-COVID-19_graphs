package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/influxdata/influxdb/client"
	"github.com/metakeule/fmtdate"
)

type JsonData struct {
	Data                   string  `json:"data"`
	Stato                  string  `json:"stato"`
	CodiceRegione          int     `json:"codice_regione"`
	DenominazioneRegione   string  `json:"denominazione_regione"`
	CodiceProvincia        int     `json:"codice_provincia"`
	DenominazioneProvincia string  `json:"denominazione_provincia"`
	SiglaProvincia         string  `json:"sigla_provincia"`
	Lat                    float64 `json:"lat"`
	Long                   float64 `json:"long"`
	TotaleCasi             int     `json:"totale_casi"`
	Datetime               time.Time
}

const andamentoProvince string = "https://raw.githubusercontent.com/pcm-dpc/COVID-19/master/dati-json/dpc-covid19-ita-province.json"
const andamentoNazionale string = "https://raw.githubusercontent.com/pcm-dpc/COVID-19/master/dati-json/dpc-covid19-ita-andamento-nazionale.json"
const dbName string = "MyDB"

func main() {

	var (
		con          *client.Client // Client for push data into InfluxDB
		host         *url.URL       // Host related to the InfluxDB instance
		httpResponse *http.Response // Response related to the HTTP request
		jsonData     []JsonData     // Data retrieved from json
		touscanyData []JsonData     // Data filtered only for touscany
		err          error
	)

	// Initialize the URL for the InfluxDB instance
	if host, err = url.Parse("http://localhost:8086"); err != nil {
		panic(err)
	}
	// Initialize the InfluxDB client
	if con, err = client.NewClient(client.Config{URL: *host}); err != nil {
		panic(err)
	}
	// Verify that InfluxDB is available
	if _, _, err = con.Ping(); err != nil {
		panic(err)
	}



	touscanyData = retrieveProvinceData(httpResponse, jsonData, "Toscana", andamentoProvince)
	dbResponse := saveInfluxProvinceData(touscanyData, con)
	fmt.Printf("%+v\n", dbResponse)
}

func saveInfluxProvinceData(touscanyData []JsonData, con *client.Client) *client.Response {
	var dbResponse *client.Response // Response related to the data pushed into InfluxDB
	var err error

	// Initialize the list of event that have to be pushed into InfluxDB
	pts := make([]client.Point, len(touscanyData))
	for i := range touscanyData {
		fmt.Println("Case: ", touscanyData[i])
		pts[i] = client.Point{
			Measurement: "all_touscany_case",
			Tags:        nil,
			Time:        touscanyData[i].Datetime,
			Fields:      map[string]interface{}{touscanyData[i].DenominazioneProvincia: touscanyData[i].TotaleCasi}}
	}

	bps := client.BatchPoints{Points: pts, Database: dbName}

	if dbResponse, err = con.Write(bps); err != nil {
		panic(err)
	}
	return dbResponse
}

func retrieveProvinceData(httpResponse *http.Response, jsonData []JsonData, regionName, urlPath string) []JsonData {
	var err error

	// Retrieve the fresh data related to covid-19
	if httpResponse, err = http.Get(urlPath); err != nil {
		panic(err)
	}

	defer httpResponse.Body.Close()

	decoder := json.NewDecoder(httpResponse.Body)
	// Decode the json into the jsonData array
	if err = decoder.Decode(&jsonData); err != nil {
		panic(err)
	}
	_ = httpResponse.Body.Close()

	fmt.Printf("Retrieved %d data\n", len(jsonData))

	// Set the local time
	if loc, err := time.LoadLocation("Europe/Rome"); err != nil {
		panic(err)
	} else {
		time.Local = loc
	}

	// Filtering the data that are only related to the Touscany region
	return filterCasesForRegion(jsonData, regionName)
}

func filterCasesForRegion(jsonData []JsonData, regionName string) []JsonData {
	var touscanyData []JsonData
	var err error
	for i := range jsonData {
		if jsonData[i].DenominazioneRegione == regionName {
			if jsonData[i].TotaleCasi > 0 {
				var t time.Time
				// Parse the time into a standard one
				if t, err = fmtdate.Parse("YYYY-MM-DD hh:mm:ss", jsonData[i].Data); err != nil {
					panic(err)
				}
				jsonData[i].Datetime = t
				touscanyData = append(touscanyData, jsonData[i])
			}
		}
	}
	return touscanyData
}
