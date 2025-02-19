// Inspired by https://github.com/codingsince1985/geo-golang
package geocoder

import (
	"cmp"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/codingsince1985/geo-golang"
	"github.com/google/go-querystring/query"
)

var (
	c               *client
	ErrClientNotSet = errors.New("geocoder: client not set")
)

const requestInterval = time.Second

type client struct {
	url         string
	client      http.Client
	logger      *slog.Logger
	lastRequest time.Time
	userAgent   string
}

type Query struct {
	Lat    float64 `url:"lat"`
	Lon    float64 `url:"lon"`
	Format string  `url:"format"`
}

type result struct {
	PlaceID     int      `json:"place_id"`
	Licence     string   `json:"licence"`
	OsmType     string   `json:"osm_type"`
	OsmID       int      `json:"osm_id"`
	Lat         string   `json:"lat"`
	Lon         string   `json:"lon"`
	PlaceRank   int      `json:"place_rank"`
	Category    string   `json:"category"`
	Type        string   `json:"type"`
	Importance  float64  `json:"importance"`
	Addresstype string   `json:"addresstype"`
	DisplayName string   `json:"display_name"`
	Name        string   `json:"name"`
	Address     *address `json:"address"`
	Boundingbox []string `json:"boundingbox"`
}

type address struct {
	HouseNumber   string `json:"house_number"`
	Road          string `json:"road"`
	Pedestrian    string `json:"pedestrian"`
	Footway       string `json:"footway"`
	Cycleway      string `json:"cycleway"`
	Highway       string `json:"highway"`
	Path          string `json:"path"`
	Suburb        string `json:"suburb"`
	City          string `json:"city"`
	Town          string `json:"town"`
	Village       string `json:"village"`
	Hamlet        string `json:"hamlet"`
	County        string `json:"county"`
	Country       string `json:"country"`
	CountryCode   string `json:"country_code"`
	State         string `json:"state"`
	StateDistrict string `json:"state_district"`
	Postcode      string `json:"postcode"`
}

func SetClient(l *slog.Logger, ua string) {
	c = &client{
		url:       "https://nominatim.openstreetmap.org/reverse",
		userAgent: ua,
		client:    http.Client{},
		logger:    l,
	}
}

func Lookup(q Query) (*geo.Address, error) {
	if c == nil {
		return nil, ErrClientNotSet
	}

	v, err := query.Values(q)
	if err != nil {
		return nil, err
	}

	if !c.lastRequest.IsZero() && time.Since(c.lastRequest) < requestInterval {
		c.logger.Warn("Rate limited - waiting " + requestInterval.String())
		time.Sleep(requestInterval)
	}

	req, err := http.NewRequest(http.MethodGet, c.url+"?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)

	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	c.lastRequest = time.Now()
	r := result{}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	return r.ToAddress(), nil
}

func (r result) ToAddress() *geo.Address {
	return &geo.Address{
		FormattedAddress: r.DisplayName,
		HouseNumber:      r.Address.HouseNumber,
		Street:           r.Address.Street(),
		Postcode:         r.Address.Postcode,
		City:             r.Address.Locality(),
		Suburb:           r.Address.Suburb,
		State:            r.Address.State,
		Country:          r.Address.Country,
		CountryCode:      strings.ToUpper(r.Address.CountryCode),
	}
}

func (a address) Locality() string {
	return cmp.Or(
		a.City, a.Town, a.Village, a.Hamlet,
	)
}

func (a address) Street() string {
	return cmp.Or(
		a.Road, a.Pedestrian, a.Path, a.Cycleway, a.Footway, a.Highway,
	)
}
