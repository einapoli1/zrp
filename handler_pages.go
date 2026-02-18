package main

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Page handlers for server-rendered templates

func pageLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		render(w, []string{"templates/login.html"}, "login", PageData{})
		return
	}
	// POST - handle form login
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")

	var id int
	var passwordHash, displayName, role string
	var active int
	err := db.QueryRow("SELECT id, password_hash, display_name, role, active FROM users WHERE username = ?", username).
		Scan(&id, &passwordHash, &displayName, &role, &active)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Invalid username or password"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Invalid username or password"))
		return
	}

	if active == 0 {
		w.WriteHeader(403)
		w.Write([]byte("Account deactivated"))
		return
	}

	// Create session
	token := generateToken()
	db.Exec("DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP")
	expires := time.Now().Add(24 * 60 * 60 * 1e9) // 24h
	db.Exec("INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)",
		token, id, expires.Format("2006-01-02 15:04:05"))
	db.Exec("UPDATE users SET last_login = CURRENT_TIMESTAMP WHERE id = ?", id)

	http.SetCookie(w, &http.Cookie{
		Name:     "zrp_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
	})

	w.Header().Set("HX-Redirect", "/dashboard")
	w.WriteHeader(200)
}

func pageLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("zrp_session")
	if err == nil {
		db.Exec("DELETE FROM sessions WHERE token = ?", cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "zrp_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func pageDashboard(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	d := DashboardData{}
	db.QueryRow("SELECT COUNT(*) FROM ecos WHERE status NOT IN ('implemented','rejected')").Scan(&d.OpenECOs)
	db.QueryRow("SELECT COUNT(*) FROM inventory WHERE qty_on_hand <= reorder_point AND reorder_point > 0").Scan(&d.LowStock)
	db.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE status NOT IN ('received','cancelled')").Scan(&d.OpenPOs)
	db.QueryRow("SELECT COUNT(*) FROM work_orders WHERE status IN ('open','in_progress')").Scan(&d.ActiveWOs)
	db.QueryRow("SELECT COUNT(*) FROM ncrs WHERE status NOT IN ('resolved','closed')").Scan(&d.OpenNCRs)
	db.QueryRow("SELECT COUNT(*) FROM rmas WHERE status NOT IN ('closed')").Scan(&d.OpenRMAs)
	db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&d.TotalDevices)
	cats, _, _ := loadPartsFromDir()
	for _, p := range cats {
		d.TotalParts += len(p)
	}

	data := PageData{
		Title:     "Dashboard",
		ActiveNav: "dashboard",
		User:      user,
		Dashboard: d,
	}

	if isHTMX(r) {
		renderPartial(w, []string{"templates/dashboard.html", "templates/partials/dashboard/kpi-cards.html"}, "content", data)
		return
	}
	render(w, []string{"templates/dashboard.html", "templates/partials/dashboard/kpi-cards.html"}, "layout", data)
}

func pageDashboardKPI(w http.ResponseWriter, r *http.Request) {
	d := DashboardData{}
	db.QueryRow("SELECT COUNT(*) FROM ecos WHERE status NOT IN ('implemented','rejected')").Scan(&d.OpenECOs)
	db.QueryRow("SELECT COUNT(*) FROM inventory WHERE qty_on_hand <= reorder_point AND reorder_point > 0").Scan(&d.LowStock)
	db.QueryRow("SELECT COUNT(*) FROM purchase_orders WHERE status NOT IN ('received','cancelled')").Scan(&d.OpenPOs)
	db.QueryRow("SELECT COUNT(*) FROM work_orders WHERE status IN ('open','in_progress')").Scan(&d.ActiveWOs)
	db.QueryRow("SELECT COUNT(*) FROM ncrs WHERE status NOT IN ('resolved','closed')").Scan(&d.OpenNCRs)
	db.QueryRow("SELECT COUNT(*) FROM rmas WHERE status NOT IN ('closed')").Scan(&d.OpenRMAs)
	db.QueryRow("SELECT COUNT(*) FROM devices").Scan(&d.TotalDevices)
	cats, _, _ := loadPartsFromDir()
	for _, p := range cats {
		d.TotalParts += len(p)
	}
	renderPartial(w, []string{"templates/partials/dashboard/kpi-cards.html"}, "kpi-cards", PageData{Dashboard: d})
}

func pageDashboardActivity(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, username, action, module, record_id, summary, created_at FROM audit_log ORDER BY created_at DESC LIMIT 15")
	if err != nil {
		renderPartial(w, []string{"templates/partials/dashboard/activity-feed.html"}, "activity-feed", PageData{})
		return
	}
	defer rows.Close()
	var activities []AuditEntry
	for rows.Next() {
		var a AuditEntry
		rows.Scan(&a.ID, &a.Username, &a.Action, &a.Module, &a.RecordID, &a.Summary, &a.CreatedAt)
		activities = append(activities, a)
	}
	renderPartial(w, []string{"templates/partials/dashboard/activity-feed.html"}, "activity-feed", PageData{Activities: activities})
}

func pagePartsList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	cats, _, _ := loadPartsFromDir()
	q := strings.ToLower(r.URL.Query().Get("q"))
	category := r.URL.Query().Get("category")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit := 50

	var all []Part
	if category != "" {
		all = cats[category]
	} else {
		for _, p := range cats {
			all = append(all, p...)
		}
	}

	if q != "" {
		var filtered []Part
		for _, p := range all {
			if strings.Contains(strings.ToLower(p.IPN), q) {
				filtered = append(filtered, p)
				continue
			}
			for _, v := range p.Fields {
				if strings.Contains(strings.ToLower(v), q) {
					filtered = append(filtered, p)
					break
				}
			}
		}
		all = filtered
	}

	total := len(all)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	if all == nil {
		all = []Part{}
	}

	// Categories for filter
	var catList []Category
	for name, parts := range cats {
		catList = append(catList, Category{ID: name, Name: name, Count: len(parts)})
	}

	extra := ""
	if q != "" {
		extra += "&q=" + q
	}
	if category != "" {
		extra += "&category=" + category
	}

	data := PageData{
		Title:      "Parts",
		ActiveNav:  "parts",
		User:       user,
		Parts:      all[start:end],
		Categories: catList,
		Query:      r.URL.Query().Get("q"),
		Category:   category,
		Total:      total,
		Pagination: makePagination(page, total, limit, "/parts", "#parts-table", extra),
	}

	if isHTMX(r) {
		renderPartial(w, []string{"templates/parts/list.html", "templates/partials/pagination.html"}, "parts-table", data)
		return
	}
	render(w, []string{"templates/parts/list.html", "templates/partials/pagination.html"}, "layout", data)
}

func pagePartDetail(w http.ResponseWriter, r *http.Request, ipn string) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	cats, _, _ := loadPartsFromDir()
	var found Part
	for _, parts := range cats {
		for _, p := range parts {
			if p.IPN == ipn {
				found = p
				break
			}
		}
	}
	if found.IPN == "" {
		http.Error(w, "Part not found", 404)
		return
	}

	upper := strings.ToUpper(ipn)
	hasBOM := strings.HasPrefix(upper, "PCA-") || strings.HasPrefix(upper, "ASY-")

	data := PageData{
		Title:     "Part: " + ipn,
		ActiveNav: "parts",
		User:      user,
		Part:      found,
		HasBOM:    hasBOM,
	}

	render(w, []string{"templates/parts/detail.html"}, "layout", data)
}

func pageECOsList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	status := r.URL.Query().Get("status")
	query := "SELECT id,title,description,status,priority,COALESCE(affected_ipns,''),created_by,created_at,updated_at FROM ecos"
	var args []interface{}
	if status != "" {
		query += " WHERE status=?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var ecos []ECO
	for rows.Next() {
		var e ECO
		rows.Scan(&e.ID, &e.Title, &e.Description, &e.Status, &e.Priority, &e.AffectedIPNs, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt)
		ecos = append(ecos, e)
	}
	if ecos == nil {
		ecos = []ECO{}
	}

	data := PageData{
		Title:     "ECOs",
		ActiveNav: "ecos",
		User:      user,
		ECOs:      ecos,
		Status:    status,
	}

	if isHTMX(r) {
		renderPartial(w, []string{"templates/ecos/list.html"}, "ecos-table", data)
		return
	}
	render(w, []string{"templates/ecos/list.html"}, "layout", data)
}

func pageECODetail(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var e ECO
	var aa, ab sql.NullString
	err := db.QueryRow("SELECT id,title,description,status,priority,COALESCE(affected_ipns,''),created_by,created_at,updated_at,approved_at,approved_by FROM ecos WHERE id=?", id).
		Scan(&e.ID, &e.Title, &e.Description, &e.Status, &e.Priority, &e.AffectedIPNs, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &aa, &ab)
	if err != nil {
		http.Error(w, "ECO not found", 404)
		return
	}
	e.ApprovedAt = sp(aa)
	e.ApprovedBy = sp(ab)

	var ipns []string
	if e.AffectedIPNs != "" {
		for _, s := range strings.Split(e.AffectedIPNs, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				ipns = append(ipns, s)
			}
		}
	}

	data := PageData{
		Title:        "ECO: " + e.ID,
		ActiveNav:    "ecos",
		User:         user,
		ECO:          e,
		AffectedIPNs: ipns,
	}

	if isHTMX(r) {
		renderPartial(w, []string{"templates/dashboard.html", "templates/partials/dashboard/kpi-cards.html"}, "content", data)
		return
	}
	render(w, []string{"templates/ecos/detail.html"}, "layout", data)
}

func pageECONew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := PageData{
		Title:     "New ECO",
		ActiveNav: "ecos",
		User:      user,
		ECO:       ECO{},
	}
	render(w, []string{"templates/ecos/form.html"}, "layout", data)
}

func pageECOCreate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	title := r.FormValue("title")
	description := r.FormValue("description")
	priority := r.FormValue("priority")
	status := r.FormValue("status")
	affectedIPNs := r.FormValue("affected_ipns")

	id := nextID("ECO", "ecos", 3)
	if status == "" {
		status = "draft"
	}
	if priority == "" {
		priority = "normal"
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	username := getUsername(r)

	_, err := db.Exec("INSERT INTO ecos (id,title,description,status,priority,affected_ipns,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)",
		id, title, description, status, priority, affectedIPNs, username, now, now)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	logAudit(db, username, "created", "eco", id, "Created "+id+": "+title)

	w.Header().Set("HX-Redirect", "/ecos/"+id)
	w.WriteHeader(200)
}

func pageECOApprove(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	username := getUsername(r)
	db.Exec("UPDATE ecos SET status='approved',approved_at=?,approved_by=?,updated_at=? WHERE id=?", now, username, now, id)
	logAudit(db, username, "approved", "eco", id, "Approved "+id)
	go emailOnECOApproved(id)

	// Re-render detail
	pageECODetail(w, r, id)
}

func pageECOImplement(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	username := getUsername(r)
	db.Exec("UPDATE ecos SET status='implemented',updated_at=? WHERE id=?", now, id)
	logAudit(db, username, "implemented", "eco", id, "Implemented "+id)

	pageECODetail(w, r, id)
}

// Quotes page handlers
func pageQuotesList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	query := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if pi, err := strconv.Atoi(p); err == nil && pi > 0 {
			page = pi
		}
	}
	limit := 50

	// Build query
	sqlQuery := "SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes WHERE 1=1"
	var args []interface{}

	if query != "" {
		sqlQuery += " AND (id LIKE ? OR customer LIKE ?)"
		args = append(args, "%"+query+"%", "%"+query+"%")
	}
	if status != "" {
		sqlQuery += " AND status = ?"
		args = append(args, status)
	}

	sqlQuery += " ORDER BY created_at DESC"

	// Count total
	countQuery := strings.Replace(sqlQuery, "SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes", "SELECT COUNT(*) FROM quotes", 1)
	var total int
	db.QueryRow(countQuery, args...).Scan(&total)

	// Get quotes for current page
	start := (page - 1) * limit
	sqlQuery += " LIMIT ? OFFSET ?"
	args = append(args, limit, start)

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		var q Quote
		var aa sql.NullString
		rows.Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
		q.AcceptedAt = sp(aa)
		quotes = append(quotes, q)
	}
	if quotes == nil {
		quotes = []Quote{}
	}

	// Build pagination extra params
	extra := ""
	if query != "" {
		extra += "&q=" + query
	}
	if status != "" {
		extra += "&status=" + status
	}

	data := PageData{
		Title:      "Quotes",
		ActiveNav:  "quotes",
		User:       user,
		Quotes:     quotes,
		Query:      query,
		Status:     status,
		Total:      total,
		Pagination: makePagination(page, total, limit, "/quotes", "#quotes-table", extra),
	}

	pageFiles := []string{"templates/quotes/list.html"}
	if isHTMX(r) {
		renderPartial(w, pageFiles, "quotes-table", data)
		return
	}
	render(w, pageFiles, "layout", data)
}

func pageQuoteDetail(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var q Quote
	var aa sql.NullString
	err := db.QueryRow("SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes WHERE id=?", id).
		Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	q.AcceptedAt = sp(aa)

	rows, _ := db.Query("SELECT id,quote_id,ipn,COALESCE(description,''),qty,COALESCE(unit_price,0),COALESCE(notes,'') FROM quote_lines WHERE quote_id=?", id)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var l QuoteLine
			rows.Scan(&l.ID, &l.QuoteID, &l.IPN, &l.Description, &l.Qty, &l.UnitPrice, &l.Notes)
			q.Lines = append(q.Lines, l)
		}
	}
	if q.Lines == nil {
		q.Lines = []QuoteLine{}
	}

	data := PageData{
		Title:     "Quote " + q.ID,
		ActiveNav: "quotes",
		User:      user,
		Quote:     q,
	}

	pageFiles := []string{"templates/quotes/detail.html"}
	render(w, pageFiles, "layout", data)
}

func pageQuoteNew(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := PageData{
		User: user,
	}

	pageFiles := []string{"templates/quotes/form.html"}
	renderPartial(w, pageFiles, "quote-form", data)
}

func pageQuoteEdit(w http.ResponseWriter, r *http.Request, id string) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var q Quote
	var aa sql.NullString
	err := db.QueryRow("SELECT id,customer,status,COALESCE(notes,''),created_at,COALESCE(valid_until,''),accepted_at FROM quotes WHERE id=?", id).
		Scan(&q.ID, &q.Customer, &q.Status, &q.Notes, &q.CreatedAt, &q.ValidUntil, &aa)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	q.AcceptedAt = sp(aa)

	data := PageData{
		User:  user,
		Quote: q,
	}

	pageFiles := []string{"templates/quotes/form.html"}
	renderPartial(w, pageFiles, "quote-form", data)
}

// Calendar page handlers
func pageCalendar(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")
	now := time.Now()
	year := now.Year()
	month := int(now.Month())
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			year = y
		}
	}
	if monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	monthNames := []string{"", "January", "February", "March", "April", "May", "June",
		"July", "August", "September", "October", "November", "December"}

	// Get events for this month
	startDate := fmt.Sprintf("%04d-%02d-01", year, month)
	endDate := fmt.Sprintf("%04d-%02d-31", year, month)

	var events []CalendarEvent

	// Work Orders
	rows, err := db.Query(`SELECT id, COALESCE(notes,''), 
		CASE WHEN completed_at IS NOT NULL THEN completed_at 
		ELSE datetime(created_at, '+30 days') END as due_date,
		assembly_ipn, qty
		FROM work_orders 
		WHERE CASE WHEN completed_at IS NOT NULL THEN completed_at 
		ELSE datetime(created_at, '+30 days') END BETWEEN ? AND ?`,
		startDate, endDate+" 23:59:59")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, notes, dueDate, assemblyIPN string
			var qty int
			rows.Scan(&id, &notes, &dueDate, &assemblyIPN, &qty)
			if len(dueDate) >= 10 {
				dueDate = dueDate[:10]
			}
			title := fmt.Sprintf("Build %s ×%d", assemblyIPN, qty)
			if notes != "" {
				title = notes
			}
			events = append(events, CalendarEvent{Date: dueDate, Type: "workorder", ID: id, Title: title, Color: "blue"})
		}
	}

	// Purchase Orders
	rows2, err := db.Query(`SELECT id, COALESCE(notes,''), expected_date FROM purchase_orders WHERE expected_date BETWEEN ? AND ?`, startDate, endDate)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var id, notes, expDate string
			rows2.Scan(&id, &notes, &expDate)
			if len(expDate) >= 10 {
				expDate = expDate[:10]
			}
			title := "PO expected delivery"
			if notes != "" {
				title = notes
			}
			events = append(events, CalendarEvent{Date: expDate, Type: "po", ID: id, Title: title, Color: "green"})
		}
	}

	// Quotes
	rows3, err := db.Query(`SELECT id, customer, valid_until FROM quotes WHERE valid_until BETWEEN ? AND ?`, startDate, endDate)
	if err == nil {
		defer rows3.Close()
		for rows3.Next() {
			var id, customer, validUntil string
			rows3.Scan(&id, &customer, &validUntil)
			if len(validUntil) >= 10 {
				validUntil = validUntil[:10]
			}
			title := fmt.Sprintf("Quote %s expires", id)
			if customer != "" {
				title = fmt.Sprintf("Quote for %s expires", customer)
			}
			events = append(events, CalendarEvent{Date: validUntil, Type: "quote", ID: id, Title: title, Color: "orange"})
		}
	}

	// Group events by date
	eventsByDate := make(map[string][]CalendarEvent)
	for _, event := range events {
		eventsByDate[event.Date] = append(eventsByDate[event.Date], event)
	}

	// Build calendar grid
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	
	// Start calendar on Sunday before first day
	startOfCalendar := firstDay.AddDate(0, 0, -int(firstDay.Weekday()))
	
	var weeks [][]CalendarDay
	current := startOfCalendar
	today := now.Format("2006-01-02")
	
	for weekNum := 0; weekNum < 6; weekNum++ {
		var week []CalendarDay
		for dayNum := 0; dayNum < 7; dayNum++ {
			dayEvents := eventsByDate[current.Format("2006-01-02")]
			day := CalendarDay{
				Day:     current.Day(),
				InMonth: current.Month() == time.Month(month),
				Today:   current.Format("2006-01-02") == today,
				Events:  dayEvents,
			}
			week = append(week, day)
			current = current.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
		// Break if we've covered the whole month
		if current.Month() != time.Month(month) && current.Day() > 7 {
			break
		}
	}

	data := PageData{
		Title:        "Calendar",
		ActiveNav:    "calendar",
		User:         user,
		Year:         year,
		Month:        month,
		MonthName:    monthNames[month],
		CalendarData: CalendarPageData{Weeks: weeks},
	}

	pageFiles := []string{"templates/calendar/index.html"}
	if isHTMX(r) {
		renderPartial(w, pageFiles, "calendar-content", data)
		return
	}
	render(w, pageFiles, "layout", data)
}

// Reports page handlers
func pageReportsList(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	data := PageData{
		Title:     "Reports",
		ActiveNav: "reports",
		User:      user,
	}

	pageFiles := []string{"templates/reports/list.html"}
	render(w, pageFiles, "layout", data)
}

func pageReportInventoryValuation(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Get report data from the existing API handler
	rows, err := db.Query(`
		SELECT i.ipn, COALESCE(i.description,''), COALESCE(i.mpn,''), i.qty_on_hand,
			COALESCE((SELECT pl.unit_price FROM po_lines pl JOIN purchase_orders po ON pl.po_id=po.id
				WHERE pl.ipn=i.ipn ORDER BY po.created_at DESC LIMIT 1), 0) as unit_price,
			COALESCE((SELECT pl.po_id FROM po_lines pl JOIN purchase_orders po ON pl.po_id=po.id
				WHERE pl.ipn=i.ipn ORDER BY po.created_at DESC LIMIT 1), '') as po_ref
		FROM inventory i ORDER BY i.ipn`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	catMap := map[string][]InvValuationItem{}
	var catOrder []string
	for rows.Next() {
		var item InvValuationItem
		var mpn string
		rows.Scan(&item.IPN, &item.Desc, &mpn, &item.QtyOnHand, &item.UnitPrice, &item.PORef)
		item.Subtotal = item.QtyOnHand * item.UnitPrice
		// Derive category from IPN prefix
		item.Category = ipnCategory(item.IPN)
		if _, ok := catMap[item.Category]; !ok {
			catOrder = append(catOrder, item.Category)
		}
		catMap[item.Category] = append(catMap[item.Category], item)
	}

	report := InvValuationReport{}
	for _, cat := range catOrder {
		items := catMap[cat]
		grp := InvValuationGroup{Category: cat, Items: items}
		for _, it := range items {
			grp.Subtotal += it.Subtotal
		}
		report.GrandTotal += grp.Subtotal
		report.Groups = append(report.Groups, grp)
	}

	data := PageData{
		Title:      "Inventory Valuation Report",
		ActiveNav:  "reports",
		User:       user,
		ReportData: report,
	}

	pageFiles := []string{"templates/reports/inventory-valuation.html"}
	if isHTMX(r) {
		renderPartial(w, pageFiles, "inventory-valuation-report", data)
		return
	}
	render(w, pageFiles, "layout", data)
}

func pageReportOpenECOs(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.Query(`SELECT id, title, status, priority, created_by, created_at FROM ecos WHERE status IN ('draft','review') ORDER BY
		CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'normal' THEN 2 WHEN 'low' THEN 3 ELSE 4 END, created_at`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []OpenECOItem
	now := time.Now()
	for rows.Next() {
		var e OpenECOItem
		rows.Scan(&e.ID, &e.Title, &e.Status, &e.Priority, &e.CreatedBy, &e.CreatedAt)
		if t, err := time.Parse("2006-01-02 15:04:05", e.CreatedAt); err == nil {
			e.AgeDays = int(now.Sub(t).Hours() / 24)
		}
		items = append(items, e)
	}
	if items == nil {
		items = []OpenECOItem{}
	}

	data := PageData{
		Title:      "Open ECOs Report",
		ActiveNav:  "reports",
		User:       user,
		ReportData: items,
	}

	pageFiles := []string{"templates/reports/open-ecos.html"}
	if isHTMX(r) {
		renderPartial(w, pageFiles, "open-ecos-report", data)
		return
	}
	render(w, pageFiles, "layout", data)
}

func pageReportWOThroughput(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	days := 30
	if d := r.URL.Query().Get("days"); d != "" {
		if v, err := strconv.Atoi(d); err == nil && (v == 30 || v == 60 || v == 90) {
			days = v
		}
	}
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")

	rows, err := db.Query(`SELECT status, started_at, completed_at FROM work_orders WHERE completed_at IS NOT NULL AND completed_at >= ?`, since)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	report := WOThroughputReport{Days: days, CountByStatus: map[string]int{}}
	var totalCycleHours float64
	cycleCount := 0
	for rows.Next() {
		var status string
		var startedAt, completedAt *string
		rows.Scan(&status, &startedAt, &completedAt)
		report.CountByStatus[status]++
		report.TotalCompleted++
		if startedAt != nil && completedAt != nil {
			if st, err1 := time.Parse("2006-01-02 15:04:05", *startedAt); err1 == nil {
				if ct, err2 := time.Parse("2006-01-02 15:04:05", *completedAt); err2 == nil {
					totalCycleHours += ct.Sub(st).Hours()
					cycleCount++
				}
			}
		}
	}
	if cycleCount > 0 {
		report.AvgCycleTimeDays = math.Round(totalCycleHours/float64(cycleCount)/24*100) / 100
	}

	data := PageData{
		Title:      "WO Throughput Report",
		ActiveNav:  "reports",
		User:       user,
		ReportData: report,
		Days:       days,
	}

	pageFiles := []string{"templates/reports/wo-throughput.html"}
	if isHTMX(r) {
		renderPartial(w, pageFiles, "wo-throughput-report", data)
		return
	}
	render(w, pageFiles, "layout", data)
}

func pageReportLowStock(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	rows, err := db.Query(`SELECT ipn, COALESCE(description,''), qty_on_hand, reorder_point, reorder_qty FROM inventory WHERE qty_on_hand < reorder_point AND reorder_point > 0 ORDER BY (reorder_point - qty_on_hand) DESC`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	var items []LowStockItem
	for rows.Next() {
		var it LowStockItem
		rows.Scan(&it.IPN, &it.Description, &it.QtyOnHand, &it.ReorderPoint, &it.ReorderQty)
		it.SuggestedOrder = it.ReorderQty
		if it.SuggestedOrder == 0 {
			it.SuggestedOrder = it.ReorderPoint - it.QtyOnHand
		}
		items = append(items, it)
	}
	if items == nil {
		items = []LowStockItem{}
	}

	data := PageData{
		Title:      "Low Stock Report",
		ActiveNav:  "reports",
		User:       user,
		ReportData: items,
	}

	pageFiles := []string{"templates/reports/low-stock.html"}
	if isHTMX(r) {
		renderPartial(w, pageFiles, "low-stock-report", data)
		return
	}
	render(w, pageFiles, "layout", data)
}

func pageReportNCRSummary(w http.ResponseWriter, r *http.Request) {
	user := getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	report := NCRSummaryReport{BySeverity: map[string]int{}, ByDefectType: map[string]int{}}

	// Open NCRs
	rows, err := db.Query(`SELECT COALESCE(severity,'unknown'), COALESCE(defect_type,'unknown') FROM ncrs WHERE status NOT IN ('closed','resolved')`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var sev, dt string
		rows.Scan(&sev, &dt)
		report.BySeverity[sev]++
		report.ByDefectType[dt]++
		report.TotalOpen++
	}

	// Avg resolve time from resolved NCRs
	rrows, err := db.Query(`SELECT created_at, resolved_at FROM ncrs WHERE resolved_at IS NOT NULL`)
	if err == nil {
		defer rrows.Close()
		var totalHours float64
		count := 0
		for rrows.Next() {
			var ca string
			var ra *string
			rrows.Scan(&ca, &ra)
			if ra != nil {
				if ct, e1 := time.Parse("2006-01-02 15:04:05", ca); e1 == nil {
					if rt, e2 := time.Parse("2006-01-02 15:04:05", *ra); e2 == nil {
						totalHours += rt.Sub(ct).Hours()
						count++
					}
				}
			}
		}
		if count > 0 {
			report.AvgResolveDays = math.Round(totalHours/float64(count)/24*100) / 100
		}
	}

	data := PageData{
		Title:      "NCR Summary Report",
		ActiveNav:  "reports",
		User:       user,
		ReportData: report,
	}

	pageFiles := []string{"templates/reports/ncr-summary.html"}
	if isHTMX(r) {
		renderPartial(w, pageFiles, "ncr-summary-report", data)
		return
	}
	render(w, pageFiles, "layout", data)
}

