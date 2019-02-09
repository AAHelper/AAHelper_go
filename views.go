package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	// "github.com/kr/pretty"

	"github.com/AAHelper/AAHelper_go/models"
	"github.com/bluele/gforms"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/kr/pretty"
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

type textInputWidget struct {
	Type  string
	Attrs map[string]string
	gforms.Widget
}

type widgetContext struct {
	Type  string
	Field gforms.FieldInterface
	Value string
	Attrs map[string]string
}

func (wg *textInputWidget) html(f gforms.FieldInterface) string {
	var buffer bytes.Buffer
	err := gforms.Template.ExecuteTemplate(&buffer, "SimpleWidget", widgetContext{
		Type:  wg.Type,
		Field: f,
		Attrs: wg.Attrs,
		Value: f.GetV().RawStr,
	})
	if err != nil {
		panic(err)
	}
	return buffer.String()
}

//TimeInputWidget Generate text input fiele: <input type="date" ...>
func TimeInputWidget() gforms.Widget {
	w := new(textInputWidget)
	w.Type = "time"
	attrs := map[string]string{}
	w.Attrs = attrs
	return w
}

type requestParams struct {
	// csrfmiddlewaretoken: d19Xqg85TAjMUkNDDU6B5kt78dNPZvKOQ2tb7ZAwHwvmVMD694wxmEJgzsdcaGH2
	Day            string
	Time           time.Time
	HoursFromStart int64
	Area           string
}

func (r *requestParams) getFutureTime() time.Time {
	now := r.Time
	then := now.Add(time.Duration(+3) * time.Hour)
	// there will be one second in the day where this is incorrect
	// And I do not care.
	if now.Hour() > 21 {
		hours := 23 - now.Hour()
		minutes := 59 - now.Minute()
		seconds := 59 - now.Second()
		then = now.Add(
			(time.Duration(hours) * time.Hour) + (time.Duration(minutes) * time.Minute) + (time.Duration(seconds) * time.Second))
	}

	return then
}

func paramsFromRequest(c *gin.Context) requestParams {
	loc, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Now().In(loc)

	// now = now.Add(time.Duration(-2) * time.Hour)
	nowt := now.Format("15:04")
	today := now.Format("Monday")
	temp := c.DefaultPostForm("HoursFromStart", "3")
	hfs, _ := strconv.ParseInt(temp, 10, 0)

	s := c.DefaultPostForm("Time", nowt)
	t, err := time.Parse("15:04", s)

	if err != nil {
		log.Println("Could not convert " + s + " to date")
	}
	rp := requestParams{
		Day:            c.DefaultPostForm("Day", today),
		Time:           t,
		HoursFromStart: hfs,
		Area:           c.DefaultPostForm("Area", "all"),
	}
	return rp

}

var cachedDayOptions [][]string
var cachedAreaOptions [][]string

func createOptionsFromRows(rows *sql.Rows, count int) [][]string {
	options := make([][]string, count+1)
	i := 1
	var option, slug string
	options[0] = make([]string, 4)
	options[0][0] = "all"
	options[0][1] = "all"
	options[0][2] = "false"
	options[0][3] = "false"

	for rows.Next() {
		rows.Scan(&option, &slug)
		options[i] = make([]string, 4)
		options[i][0] = strings.Title(strings.ToLower(option))
		options[i][1] = slug
		options[i][2] = "false"
		options[i][3] = "false"

		i++
	}
	return options
}

func createDayOptions(conn *gorm.DB, selectedOption string) func() gforms.SelectOptions {
	if cachedDayOptions == nil {
		var count int
		conn.Model(&models.Type{}).Count(&count)
		rows, err := conn.Model(&models.Type{}).Select("type, slug").Order("id asc").Rows()
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		cachedDayOptions = createOptionsFromRows(rows, count)
	}
	return func() gforms.SelectOptions {
		options := make([][]string, len(cachedDayOptions))
		copy(options, cachedDayOptions)
		for _, values := range options {
			if values[1] == selectedOption {
				values[2] = "true"
			}
		}
		return gforms.StringSelectOptions(options)
	}
}
func createAreaOptions(conn *gorm.DB, selectedOption string) func() gforms.SelectOptions {
	// conn.Select("aafinder_meetingtype.Type, aafinder_meetingtype.slug"
	if cachedAreaOptions == nil {
		var count int
		conn.Model(&models.Area{}).Count(&count)
		rows, err := conn.Model(&models.Area{}).Select("area, slug").Order("id desc").Rows()
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		cachedAreaOptions = createOptionsFromRows(rows, count)

	}
	return func() gforms.SelectOptions {
		options := make([][]string, len(cachedAreaOptions))
		copy(options, cachedAreaOptions)
		for _, values := range options {
			if values[1] == selectedOption {
				values[2] = "true"
			}
		}
		return gforms.StringSelectOptions(options)

	}
}

func createForm(conn *gorm.DB, c *gin.Context, rps requestParams) gforms.Form {
	// options := createDayOptions(conn, rps.Day)
	// areas := createAreaOptions(conn, rps.Area)
	options := createDayOptions(conn, rps.Day)
	areas := createAreaOptions(conn, rps.Area)
	form := gforms.DefineForm(
		gforms.NewFields(
			gforms.NewTextField(
				"Day",
				gforms.Validators{
					gforms.Required(),
				},
				gforms.SelectWidget(
					map[string]string{},
					options,
				),
			),
			gforms.NewDateTimeField(
				"Time",
				"15:04",
				gforms.Validators{},
				// gforms.TextInputWidget(
				// 	map[string]string{
				// 		"type": "time",
				// 	},
				// ),
				TimeInputWidget(),
			),
			gforms.NewIntegerField(
				"HoursFromStart",
				gforms.Validators{
					gforms.Required(),
				},
			),
			gforms.NewTextField(
				"Area",
				gforms.Validators{
					gforms.Required(),
				},
				gforms.SelectWidget(
					map[string]string{
						" class": "custom",
					},
					areas,
				),
			),
		),
	)
	return form
}

func replaceSingleQuote(s string) string {
	return strings.Replace(s, "'", "\\'", -1)
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
		replaceSingleQuote(m.Name),
		m.URL,
		replaceSingleQuote(m.AddressString),
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

func index(conn *gorm.DB, c *gin.Context, isPost bool) {
	meetings := []models.VirtualMeeting{}
	m := models.VirtualMeeting{}

	// TODO: Figure out a long term solution for this.
	// Setting the timezone manually like this is probably
	// not the best way to handle it long-term.
	// loc, _ := time.LoadLocation("America/Los_Angeles")
	// now := time.Now().In(loc)

	// now = now.Add(time.Duration(-2) * time.Hour)
	conn.Set("gorm:auto_preload", true)

	rps := paramsFromRequest(c)
	form := createForm(conn, c, rps)
	instance := form()
	if isPost {
		// pretty.Println("It's a post!")
		instance = form.FromRequest(c.Request)
	} else {
		// pretty.Println("NOT A POST!")
	}

	fi, exists := instance.GetField("HoursFromStart")
	if exists {
		fi.SetInitial("3")
	}

	fi, exists = instance.GetField("Time")
	if exists {
		fi.SetInitial(rps.Time.Format("15:04"))
	}

	if instance.IsValid() {
		instance.MapTo(&rps)
	}

	// pretty.Println(instance)
	// pretty.Println(instance.Errors())
	// pretty.Println(instance.CleanedData)
	// pretty.Println(cachedDayOptions)
	// pretty.Println(form.Html())
	area := models.Area{}
	if rps.Area != "all" {
		conn.FirstOrInit(&area, &models.Area{Slug: rps.Area})
	}

	if err := m.QueryWithDay(conn).
		Where("aafinder_meeting.time >= ? AND aafinder_meeting.time <= ?", rps.Time.Format("15:04"), rps.getFutureTime().Format("15:04")).
		Where("day.slug=?", strings.ToLower(rps.Day)).
		Find(&meetings).Error; err != nil {
		log.Fatal(err)
	}
	data := map[string]interface{}{
		"form":                instance,
		"area":                area,
		"show_errors":         isPost,
		"latest_meeting_list": meetings,
		"meeting_js":          makeNewMeetingsJS(meetings),
		"now":                 rps.Time,
		"hours_from":          rps.getFutureTime(),
		"today":               rps.Day,
	}
	if gin.IsDebugging() {
		pretty.Println(data)
	}
	c.Set("template", "templates/index.html")
	c.Set("data", data)
}

func locationDetail(locationID int64, conn *gorm.DB, c *gin.Context) {
	meetings := []models.VirtualMeeting{}
	m := models.VirtualMeeting{}
	location := models.Location{}
	if err := conn.Where(&models.Location{ID: locationID}).First(&location).Error; err != nil {
		log.Fatal(err)
	}
	if err := m.QueryNoDay(conn).
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

	if err := m.QueryNoDay(conn).
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
