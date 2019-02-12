package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AAHelper/AAHelper_go/models"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/mrfunyon/gforms"
	csrf "github.com/utrack/gin-csrf"
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

const defaultDay = "today"
const defaultType = "all"

type requestParams struct {
	Day            string
	Time           time.Time
	HoursFromStart int64
	Area           string
}

func (r *requestParams) getFutureTime() time.Time {
	now := r.Time
	then := now.Add(time.Duration(r.HoursFromStart) * time.Hour)
	// pretty.Println(then.Hour(), then.Minute(), then.Second())
	// there will be one second in the day where this is incorrect
	// And I do not care.
	t := now.Add(time.Duration(r.HoursFromStart) * time.Hour)
	if now.Weekday() != t.Weekday() {
		hours := 23 - now.Hour()
		minutes := 59 - now.Minute()
		seconds := 59 - now.Second()
		// pretty.Println(hours, minutes, seconds, r.HoursFromStart)
		then = now.Add(
			(time.Duration(hours) * time.Hour) + (time.Duration(minutes) * time.Minute) + (time.Duration(seconds) * time.Second))
	}

	return then
}

func (r *requestParams) getTodayIfDayIsAll() string {
	if r.Day == defaultDay {
		// TODO: Figure out a long term solution for this.
		// Setting the timezone manually like this is probably

		loc, _ := time.LoadLocation("America/Los_Angeles")
		now := time.Now().In(loc)
		return now.Format("Monday")
	}
	return r.Day
}

func paramsFromRequest(c *gin.Context) requestParams {
	// TODO: Figure out a long term solution for this.
	// Setting the timezone manually like this is probably
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
		Area:           c.DefaultPostForm("Area", defaultType),
	}
	return rp

}

var cachedDayOptions [][]string
var cachedAreaOptions [][]string

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

func createOptionsFromRows(rows *sql.Rows, count int, defaultOptionWord string) [][]string {
	options := make([][]string, count+1)
	i := 1
	var option, slug string
	options[0] = make([]string, 4)
	options[0][0] = strings.Title(defaultOptionWord)
	options[0][1] = defaultOptionWord
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
		cachedDayOptions = createOptionsFromRows(rows, count, defaultDay)
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
		cachedAreaOptions = createOptionsFromRows(rows, count, defaultType)

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
				gforms.TimeInputWidget(map[string]string{
					"type": "time",
				}),
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

func createFormAndRPS(conn *gorm.DB, c *gin.Context, isPost bool) (requestParams, *gforms.FormInstance) {
	rps := paramsFromRequest(c)
	form := createForm(conn, c, rps)
	instance := form()
	if isPost {
		instance = form.FromRequest(c.Request)
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
	if rps.Day == defaultDay {
		rps.Day = rps.getTodayIfDayIsAll()
	}
	return rps, instance
}

func index(conn *gorm.DB, c *gin.Context, isPost bool) {
	meetings := []models.VirtualMeeting{}
	m := models.VirtualMeeting{}

	conn.Set("gorm:auto_preload", true)

	area := models.Area{}

	rps, instance := createFormAndRPS(conn, c, isPost)

	if rps.Area != defaultType {
		conn.FirstOrInit(&area, &models.Area{Slug: rps.Area})
	}

	query := m.QueryWithDay(conn).
		Where("aafinder_meeting.time >= ? AND aafinder_meeting.time <= ?", rps.Time.Format("15:04"), rps.getFutureTime().Format("15:04"))

	if rps.Day != defaultDay {
		query = query.Where("day.slug=?", strings.ToLower(rps.Day))
	}

	if err := query.Find(&meetings).Error; err != nil {
		log.Fatal(err)
	}
	// user, exists := c.Get("user")
	// if !exists {
	// 	user = models.User{}
	// }
	// pretty.Println(user)

	context := c.GetStringMap("context")
	context["form"] = instance
	context["area"] = area
	context["show_errors"] = isPost
	context["latest_meeting_list"] = meetings
	context["meeting_js"] = makeNewMeetingsJS(meetings)
	context["now"] = rps.Time
	context["hours_from"] = rps.getFutureTime()
	context["today"] = rps.Day
	context["csrf_token"] = csrf.GetToken(c)
	// context["user"] = user

	// if gin.IsDebugging() {
	// 	pretty.Println(data)
	// }
	c.Set("template", "index.html")
	c.Set("data", context)
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
	_, instance := createFormAndRPS(conn, c, false)
	c.Set("template", "locations.html")

	context := c.GetStringMap("context")
	context["form"] = instance
	context["latest_meeting_list"] = meetings
	context["meeting_js"] = makeNewMeetingsJS(meetings)
	context["area"] = location
	context["index"] = false
	context["csrf_token"] = csrf.GetToken(c)
	c.Set("data", context)

}

func areaDetail(Slug string, conn *gorm.DB, c *gin.Context) {
	meetings := []models.VirtualMeeting{}
	m := models.VirtualMeeting{}
	area := models.Area{}
	_, instance := createFormAndRPS(conn, c, false)

	if err := conn.Where(&models.Area{Slug: Slug}).First(&area).Error; err != nil {
		log.Fatal(err)
	}

	if err := m.QueryNoDay(conn).
		Where("aafinder_meeting.area_id=?", area.ID).
		Find(&meetings).Error; err != nil {
		log.Fatal(err)
	}

	c.Set("template", "area.html")
	context := c.GetStringMap("context")
	context["form"] = instance
	context["latest_meeting_list"] = meetings
	context["meeting_js"] = makeNewMeetingsJS(meetings)
	context["area"] = area
	context["index"] = false
	context["csrf_token"] = csrf.GetToken(c)
	c.Set("data", context)
}
