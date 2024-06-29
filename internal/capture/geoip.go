package capture

import (
	"net"
	"strconv"

	"github.com/mosajjal/dnsmonster/internal/util"
	"github.com/oschwald/maxminddb-golang"
	log "github.com/sirupsen/logrus"
)

type MaxminddbRecord struct {
	Continent struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"continent"`
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	AutonomousSystemNumber       int    `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

// Close closes the database readers if they are open
func (config captureConfig) Close() {
	if config.dbCountryReader != nil {
		config.dbCountryReader.Close()
	}
	if config.dbCityReader != nil {
		config.dbCityReader.Close()
	}
	if config.dbASNReader != nil {
		config.dbASNReader.Close()
	}
}

// Load GeoIP databases
func (config *captureConfig) LoadGeoIP() error {
	// before to open, close all files
	// because open can be called also on reload
	config.Close()
	var err error
	if config.GeoIpCountryFile != "" {
		config.dbCountryReader, err = maxminddb.Open(config.GeoIpCountryFile)
		if err != nil {
			return err
		}
		log.Infof("GeoIP: Country database loaded (%d records)", config.dbCountryReader.Metadata.NodeCount)
	}

	if config.GeoIpASNFile != "" {
		config.dbASNReader, err = maxminddb.Open(config.GeoIpASNFile)
		if err != nil {
			return err
		}
		log.Infof("GeoIP: ASN database loaded (%d records)", config.dbASNReader.Metadata.NodeCount)
	}

	if config.GeoIpCityFile != "" {
		config.dbCityReader, err = maxminddb.Open(config.GeoIpCityFile)
		if err != nil {
			return err
		}
		log.Infof("GeoIP: City database loaded (%d records)", config.dbCityReader.Metadata.NodeCount)
	}
	return nil
}

// maxmindGeoIP returns the GeoIP information for the given IP address
func (config captureConfig) maxmindGeoIP(ip net.IP) util.GeoRecord {
	record := &MaxminddbRecord{}
	rec := util.GeoRecord{Continent: "-", CountryISOCode: "-", City: "-", ASN: "-", ASO: "-"}

	if config.dbASNReader != nil {
		err := config.dbASNReader.Lookup(ip, &record)
		if err != nil {
			log.Error(err)
		}
		rec.ASN = strconv.Itoa(record.AutonomousSystemNumber)
		rec.ASO = record.AutonomousSystemOrganization
	}

	if config.dbCityReader != nil {
		err := config.dbCityReader.Lookup(ip, &record)
		if err != nil {
			log.Error(err)
		}
		rec.City = record.City.Names["en"]
		rec.CountryISOCode = record.Country.ISOCode
		rec.Continent = record.Continent.Code

	} else if config.dbCountryReader != nil {
		err := config.dbCountryReader.Lookup(ip, &record)
		if err != nil {
			log.Error(err)
		}
		rec.CountryISOCode = record.Country.ISOCode
		rec.Continent = record.Continent.Code
	}
	return rec
}
