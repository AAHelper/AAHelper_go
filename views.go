package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	// "github.com/kr/pretty"

	"github.com/AAHelper/AAHelper_go/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
)

func chunkify(actions []string) [][]string {
	batchSize := 3
	var batches [][]string

	for batchSize < len(actions) {
		actions, batches = actions[batchSize:], append(batches, actions[0:batchSize:batchSize])
	}
	batches = append(batches, actions)
	return batches

}

func createPopUpText(m models.VirtualMeeting) string {
	var s []string
	meetingDays := m.Types
	sort.Slice(meetingDays[:], func(i, j int) bool {
		return meetingDays[i].ID < meetingDays[j].ID
	})
	for _, t := range meetingDays {
		s = append(s, t.Type)
	}
	days := ""
	for _, batch := range chunkify(s) {
		days += "<br />" + strings.Join(batch[:], ",")
	}
	// days := strings.Join(s[:], ",")
	return fmt.Sprintf(
		"<p>%s<br /><a href=\"%s\">MAP</a><br />%s%s <br />%s</p><hr />",
		m.Name,
		m.URL,
		m.AddressString,
		days,
		m.Time.Format("15:04"),
	)

}

type jsLoc struct {
	Meeting   models.VirtualMeeting
	PopUpText string
	Lat       float64
	Lon       float64
}

type meetingsJS struct {
	Meetings  []models.VirtualMeeting
	Locations map[sql.NullInt64]jsLoc
}

func makeNewMeetingsJS(meetings []models.VirtualMeeting) *meetingsJS {
	l := new(meetingsJS)
	l.Meetings = meetings
	for _, meeting := range meetings {
		if entry, ok := l.Locations[meeting.LocationID]; ok {
			entry.PopUpText += createPopUpText(meeting)
		} else {
			if l.Locations == nil {
				l.Locations = map[sql.NullInt64]jsLoc{}
			}
			l.Locations[meeting.LocationID] = jsLoc{
				Meeting:   meeting,
				PopUpText: createPopUpText(meeting),
				Lat:       meeting.Location.Lat,
				Lon:       meeting.Location.Lon,
			}
		}
	}
	return l
}

func index(conn *gorm.DB, c *gin.Context) {
	meetings := []models.VirtualMeeting{}
	m := models.VirtualMeeting{}

	now := time.Now()

	now = now.Add(time.Duration(-2) * time.Hour)
	conn.Set("gorm:auto_preload", true)
	nowt := now.Format("15:04")
	then := now.Add(time.Duration(+3) * time.Hour)
	thent := then.Format("15:04")
	if err := m.BaseQuery(conn).
		Where("aafinder_meeting.time between ? AND ?", nowt, thent).
		Where("day.slug=?", strings.ToLower(now.Weekday().String())).
		Find(&meetings).Error; err != nil {
		log.Fatal(err)
	}

	c.Set("template", "templates/index.html")
	c.Set("data", map[string]interface{}{
		"latest_meeting_list": meetings,
		"meeting_js":          makeNewMeetingsJS(meetings),
		"now":                 now,
		"hours_from":          then,
	})
}

func locationDetail(locationID int64, conn *gorm.DB, c *gin.Context) {
	meetings := []models.VirtualMeeting{}
	m := models.VirtualMeeting{}
	location := models.Location{}
	if err := conn.Where(&models.Location{ID: locationID}).First(&location).Error; err != nil {
		log.Fatal(err)
	}
	if err := m.BaseQuery(conn).
		Where("aafinder_meeting.location_id=?", location.ID).
		Find(&meetings).Error; err != nil {
		log.Fatal(err)
	}

	c.Set("template", "templates/locations.html")
	c.Set("data", map[string]interface{}{
		"latest_meeting_list": meetings,
		"meeting_js":          makeNewMeetingsJS(meetings),
		"area":                location,
		"index":               false,
	})
}

func areaDetail(Slug string, conn *gorm.DB, c *gin.Context) {
	meetings := []models.VirtualMeeting{}
	m := models.VirtualMeeting{}
	area := models.Area{}

	if err := conn.Where(&models.Area{Slug: Slug}).First(&area).Error; err != nil {
		log.Fatal(err)
	}

	if err := m.BaseQuery(conn).
		Where("aafinder_meeting.area_id=?", area.ID).
		Find(&meetings).Error; err != nil {
		log.Fatal(err)
	}

	c.Set("template", "templates/area.html")
	c.Set("data", map[string]interface{}{
		"latest_meeting_list": meetings,
		"meeting_js":          makeNewMeetingsJS(meetings),
		"area":                area,
		"index":               false,
	})
}
