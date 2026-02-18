package main

import (
	"database/sql"
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

