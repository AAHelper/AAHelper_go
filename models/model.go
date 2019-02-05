package models

//go:generate kallax gen

import (
	"context"
	"database/sql"
	"log"
	"time"

	// postgis "github.com/cridenour/go-postgis"

	"github.com/jinzhu/gorm"
	"github.com/kr/pretty"

	pqtype "github.com/mc2soft/pq-types"
	"googlemaps.github.io/maps"
	// kallax "gopkg.in/src-d/go-kallax.v1"
)

//Location is where a meeting is held
type Location struct {
	ID            int64
	address       *string
	city          *string
	state         *string
	ZipCode       *string
	AddressString string
	// Lat           float64
	// Lng           float64
	// Location      int64 `kallax:"-"`
	Location pqtype.PostGISPoint
	checked  bool `gorm:"-"`
}

//TableName of Location
func (Location) TableName() string {
	return "aafinder_location"
}

//GetName of an area, generic interface for the template
func (l Location) String() string {
	return l.AddressString
}

//BeforeSave on the location to geolocate the request
func (l *Location) BeforeSave() (err error) {
	if l.Location.Lon == 0 || l.Location.Lat == 0 {
		c, err := maps.NewClient(maps.WithAPIKey("AIzaSyCRB2jA_b4InjlQtslR5g5NO9n8dUTdJ0Q"))
		if err != nil {
			log.Fatalf("fatal error: %s", err)
		}
		r := &maps.FindPlaceFromTextRequest{
			Input: l.AddressString,
		}
		location, err := c.FindPlaceFromText(context.Background(), r)

		if err != nil {
			log.Fatalf("fatal error: %s", err)
		}
		for _, candidate := range location.Candidates {
			p := pqtype.PostGISPoint{
				Lat: candidate.Geometry.Location.Lat,
				Lon: candidate.Geometry.Location.Lng,
			}
			l.Location = p
			break
		}

		pretty.Println(location)
		l.checked = true
	}
	return nil
}

//Code is a Meeting code or Meeting Type
type Code struct {
	ID          int64
	Code        string
	Description string
}

//TableName of CodeOrType
func (Code) TableName() string {
	return "aafinder_meetingcode"
}

//Type is a Meeting code or Meeting Type
type Type struct {
	ID   int64
	Type string
	Slug string
}

//TableName of CodeOrType
func (Type) TableName() string {
	return "aafinder_meetingtype"
}

//Area is the meeting's area
type Area struct {
	ID   int64
	Area string
	Slug string
}

//TableName of Area
func (Area) TableName() string {
	return "aafinder_meetingarea"
}

//GetName of an area, generic interface for the template
func (a Area) String() string {
	return a.Area
}

//MeetingMeetingCodes M2M table for meeting->Codes
// type MeetingMeetingCodes struct {
// 	kallax.Model
// 	Meeting *Meeting    `fk:"meeting_codes"`
// 	Code    *CodeOrType `fk:"meeting_code_or_type"`
// }

// //MeetingMeetingTypes M2M table for meeting->Types
// type MeetingMeetingTypes struct {
// 	kallax.Model
// 	Meeting *Meeting    `fk:"meeting_codes"`
// 	Type    *CodeOrType `fk:"meeting_code_or_type"`
// }

// Meeting is the main thing
type Meeting struct {
	ID   int64
	Name string
	Time time.Time
	// URL          url.URL
	URL          string
	Area         Area `gorm:"auto_preload"`
	AreaID       sql.NullInt64
	Location     Location `gorm:"auto_preload"`
	LocationID   sql.NullInt64
	OrigFilename string
	RowSrc       string
	Notes        string
	Codes        []Code `gorm:"many2many:aafinder_meeting_codes;foreignkey:id;association_foreignkey:id;association_jointable_foreignkey:meetingcode_id;jointable_foreignkey:meeting_id;preload"`
	Types        []Type `gorm:"many2many:aafinder_meeting_types;foreignkey:id;association_foreignkey:id;association_jointable_foreignkey:meetingtype_id;jointable_foreignkey:meeting_id;preload"`
}

//TableName of Meeting
func (Meeting) TableName() string {
	return "aafinder_meeting"
}

//VirtualMeeting Table
type VirtualMeeting struct {
	ID            int64
	Name          string
	Time          time.Time
	URL           string
	Area          string
	AreaSlug      string
	LocationID    sql.NullInt64
	AddressString string
	Location      pqtype.PostGISPoint
	// OrigFilename  string
	// RowSrc        string
	Notes string
	Day   string
	Codes []Code `gorm:"many2many:aafinder_meeting_codes;foreignkey:id;association_foreignkey:id;association_jointable_foreignkey:meetingcode_id;jointable_foreignkey:meeting_id;preload"`
	Types []Type `gorm:"many2many:aafinder_meeting_types;foreignkey:id;association_foreignkey:id;association_jointable_foreignkey:meetingtype_id;jointable_foreignkey:meeting_id;preload"`
}

//TableName of Meeting
func (VirtualMeeting) TableName() string {
	return "aafinder_meeting"
}

//BaseQuery Creates the query base to generate a VirtualMeeting
func (VirtualMeeting) BaseQuery(conn *gorm.DB) *gorm.DB {
	return conn.Select("aafinder_meeting.id as id, name, time, url, area.area as area, area.slug as area_slug, location.id as location_id, location.address_string as address_string, location.location as location, notes, day.type as day").
		Joins("JOIN aafinder_location as location ON aafinder_meeting.location_id=location.id").
		Joins("JOIN aafinder_meetingarea as area ON aafinder_meeting.area_id=area.id").
		Joins("JOIN aafinder_meeting_types ON aafinder_meeting_types.meeting_id = aafinder_meeting.id").
		Joins("JOIN aafinder_meetingtype as day ON aafinder_meeting_types.meetingtype_id=day.id").
		Group("aafinder_meeting.id, area.area, area.slug, location.address_string, location.location, location.id, day.type").
		Order("aafinder_meeting.time, aafinder_meeting.name DESC").
		Preload("Codes").
		Preload("Types")
}

func (vm *VirtualMeeting) GetJSLocationText() {

}
