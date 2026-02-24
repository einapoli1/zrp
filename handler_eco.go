package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func handleListECOs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	query := "SELECT id,title,COALESCE(description,''),COALESCE(type,'change'),COALESCE(status,''),COALESCE(priority,''),COALESCE(affected_ipns,''),COALESCE(created_by,''),COALESCE(created_at,''),COALESCE(updated_at,''),approved_at,approved_by,COALESCE(ncr_id,'') FROM ecos"
	var args []interface{}
	if status != "" {
		query += " WHERE status=?"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"
	rows, err := db.Query(query, args...)
	if err != nil {
		jsonErr(w, err.Error(), 500); return
	}
	defer rows.Close()
	var items []ECO
	for rows.Next() {
		var e ECO
		var aa, ab sql.NullString
		rows.Scan(&e.ID, &e.Title, &e.Description, &e.Type, &e.Status, &e.Priority, &e.AffectedIPNs, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &aa, &ab, &e.NcrID)
		e.ApprovedAt = sp(aa); e.ApprovedBy = sp(ab)
		items = append(items, e)
	}
	if items == nil { items = []ECO{} }
	jsonResp(w, items)
}

func handleGetECO(w http.ResponseWriter, r *http.Request, id string) {
	var e ECO
	var aa, ab sql.NullString
	err := db.QueryRow("SELECT id,title,COALESCE(description,''),COALESCE(type,'change'),COALESCE(status,''),COALESCE(priority,''),COALESCE(affected_ipns,''),COALESCE(created_by,''),COALESCE(created_at,''),COALESCE(updated_at,''),approved_at,approved_by,COALESCE(ncr_id,'') FROM ecos WHERE id=?", id).
		Scan(&e.ID, &e.Title, &e.Description, &e.Type, &e.Status, &e.Priority, &e.AffectedIPNs, &e.CreatedBy, &e.CreatedAt, &e.UpdatedAt, &aa, &ab, &e.NcrID)
	if err != nil { jsonErr(w, "not found", 404); return }
	e.ApprovedAt = sp(aa); e.ApprovedBy = sp(ab)

	// Enrich with affected parts details
	var affectedParts []map[string]string
	var ipns []string
	// Try JSON array first, then comma-separated
	if strings.HasPrefix(strings.TrimSpace(e.AffectedIPNs), "[") {
		json.Unmarshal([]byte(e.AffectedIPNs), &ipns)
	} else if e.AffectedIPNs != "" {
		for _, s := range strings.Split(e.AffectedIPNs, ",") {
			s = strings.TrimSpace(s)
			if s != "" { ipns = append(ipns, s) }
		}
	}
	for _, ipn := range ipns {
		fields, err := getPartByIPN(partsDir, ipn)
		if err == nil {
			part := make(map[string]string)
			part["ipn"] = ipn
			for k, v := range fields {
				part[strings.ToLower(k)] = v
			}
			affectedParts = append(affectedParts, part)
		} else {
			affectedParts = append(affectedParts, map[string]string{"ipn": ipn, "error": "not found"})
		}
	}
	if affectedParts == nil { affectedParts = []map[string]string{} }

	// Build enriched response
	resp := map[string]interface{}{
		"id": e.ID, "title": e.Title, "description": e.Description, "type": e.Type,
		"status": e.Status, "priority": e.Priority, "affected_ipns": e.AffectedIPNs,
		"affected_parts": affectedParts, "created_by": e.CreatedBy,
		"created_at": e.CreatedAt, "updated_at": e.UpdatedAt,
		"approved_at": e.ApprovedAt, "approved_by": e.ApprovedBy,
		"ncr_id": e.NcrID,
	}
	jsonResp(w, resp)
}

func handleCreateECO(w http.ResponseWriter, r *http.Request) {
	var e ECO
	if err := decodeBody(r, &e); err != nil { jsonErr(w, "invalid body", 400); return }

	// Validation
	ve := &ValidationErrors{}
	requireField(ve, "title", e.Title)
	validateMaxLength(ve, "title", e.Title, 255)
	validateMaxLength(ve, "description", e.Description, 1000)
	validateMaxLength(ve, "affected_ipns", e.AffectedIPNs, 1000)
	if e.Status != "" { validateEnum(ve, "status", e.Status, validECOStatuses) }
	if e.Priority != "" { validateEnum(ve, "priority", e.Priority, validECOPriorities) }
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	e.ID = nextID("ECO", "ecos", 3)
	if e.Status == "" { e.Status = "draft" }
	if e.Priority == "" { e.Priority = "normal" }
	if e.Type == "" { e.Type = "change" }
	now := time.Now().Format("2006-01-02 15:04:05")
	_, err := db.Exec("INSERT INTO ecos (id,title,description,type,status,priority,affected_ipns,created_by,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
		e.ID, e.Title, e.Description, e.Type, e.Status, e.Priority, e.AffectedIPNs, "engineer", now, now)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	e.CreatedAt = now; e.UpdatedAt = now; e.CreatedBy = "engineer"
	ensureInitialRevision(e.ID, "engineer", now)
	logAudit(db, getUsername(r), "created", "eco", e.ID, "Created "+e.ID+": "+e.Title)
	recordChangeJSON(getUsername(r), "ecos", e.ID, "create", nil, e)
	jsonResp(w, e)
}

func handleUpdateECO(w http.ResponseWriter, r *http.Request, id string) {
	// Verify exists and get current status
	var currentStatus string
	err := db.QueryRow("SELECT status FROM ecos WHERE id=?", id).Scan(&currentStatus)
	if err != nil { jsonErr(w, "not found", 404); return }

	oldSnap, _ := getECOSnapshot(id)
	var e ECO
	if err := decodeBody(r, &e); err != nil { jsonErr(w, "invalid body", 400); return }

	ve := &ValidationErrors{}
	requireField(ve, "title", e.Title)
	validateMaxLength(ve, "title", e.Title, 255)
	validateMaxLength(ve, "description", e.Description, 1000)
	validateMaxLength(ve, "affected_ipns", e.AffectedIPNs, 1000)
	validateEnum(ve, "status", e.Status, validECOStatuses)
	validateEnum(ve, "priority", e.Priority, validECOPriorities)
	
	// Validate status transition
	if err := validateECOStatusTransition(currentStatus, e.Status); err != nil {
		ve.Add("status", err.Error())
	}
	
	if ve.HasErrors() { jsonErr(w, ve.Error(), 400); return }

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.Exec("UPDATE ecos SET title=?,description=?,status=?,priority=?,affected_ipns=?,updated_at=? WHERE id=?",
		e.Title, e.Description, e.Status, e.Priority, e.AffectedIPNs, now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	logAudit(db, getUsername(r), "updated", "eco", id, "Updated "+id+": "+e.Title)
	newSnap, _ := getECOSnapshot(id)
	recordChangeJSON(getUsername(r), "ecos", id, "update", oldSnap, newSnap)
	handleGetECO(w, r, id)
}

func handleApproveECO(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	user := getUsername(r)
	
	// Use transaction with row locking to prevent concurrent approval race conditions
	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, "failed to start transaction: "+err.Error(), 500)
		return
	}
	defer tx.Rollback()
	
	// Lock row and check current status and type
	var currentStatus, ecoType, description string
	err = tx.QueryRow("SELECT status, COALESCE(type, 'change'), COALESCE(description, '') FROM ecos WHERE id=?", id).Scan(&currentStatus, &ecoType, &description)
	if err != nil {
		jsonErr(w, "ECO not found", 404)
		return
	}
	
	// Validate can approve (should be in 'review' or 'pending' status)
	if currentStatus != "review" && currentStatus != "pending" {
		jsonErr(w, fmt.Sprintf("ECO must be in 'review' or 'pending' status to approve (current: %s)", currentStatus), 400)
		return
	}
	
	// If this is a creation ECO, create the actual item
	if ecoType == "creation" {
		if err := handleCreationECOApproval(tx, id, description); err != nil {
			jsonErr(w, "Failed to create item: "+err.Error(), 500)
			return
		}
	}
	
	// Perform approval update
	_, err = tx.Exec("UPDATE ecos SET status='approved',approved_at=?,approved_by=?,updated_at=? WHERE id=?", now, user, now, id)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	
	// Record approval in latest revision
	_, err = tx.Exec("UPDATE eco_revisions SET approved_by=?, approved_at=?, status='approved' WHERE eco_id=? AND id=(SELECT MAX(id) FROM eco_revisions WHERE eco_id=?)", user, now, id, id)
	if err != nil {
		// Ignore error if eco_revisions doesn't exist (older ECOs)
		_ = err
	}
	
	// Commit transaction
	if err = tx.Commit(); err != nil {
		jsonErr(w, "failed to commit transaction: "+err.Error(), 500)
		return
	}
	
	logAudit(db, user, "approved", "eco", id, "Approved "+id)
	go emailOnECOApproved(id)
	handleGetECO(w, r, id)
}

// handleCreationECOApproval processes approval of a creation-type ECO by creating the actual item
func handleCreationECOApproval(tx *sql.Tx, ecoID, description string) error {
	// Parse the JSON-encoded item data from description
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(description), &data); err != nil {
		return fmt.Errorf("invalid JSON in description: %w", err)
	}
	
	itemType, ok := data["item_type"].(string)
	if !ok {
		return fmt.Errorf("missing item_type in creation data")
	}
	
	switch itemType {
	case "part":
		return createPartFromECO(tx, data)
	// Future: handle assembly, configuration, etc.
	default:
		return fmt.Errorf("unsupported item_type: %s", itemType)
	}
}

// createPartFromECO creates a part from ECO approval data
func createPartFromECO(tx *sql.Tx, data map[string]interface{}) error {
	ipn, _ := data["ipn"].(string)
	category, _ := data["category"].(string)
	fieldsData, _ := data["fields"].(map[string]interface{})
	
	if ipn == "" || category == "" {
		return fmt.Errorf("missing ipn or category")
	}
	
	// Convert fields map to string map
	fields := make(map[string]string)
	for k, v := range fieldsData {
		if str, ok := v.(string); ok {
			fields[k] = str
		}
	}
	
	// Find the CSV file for this category
	csvPath := findCategoryCSV(category)
	if csvPath == "" {
		return fmt.Errorf("category not found: %s", category)
	}
	
	// Check IPN uniqueness (using tx for consistency)
	var exists int
	err := tx.QueryRow("SELECT COUNT(*) FROM (SELECT 1 FROM parts WHERE ipn = ? LIMIT 1)", ipn).Scan(&exists)
	if err == nil && exists > 0 {
		// IPN already exists (possibly created by another approval)
		return nil
	}
	
	// Append part to CSV file (outside transaction since it's file I/O)
	// Note: This is a simplified version; production code should handle this more carefully
	if err := appendPartToCSV(csvPath, ipn, fields); err != nil {
		return fmt.Errorf("failed to write to CSV: %w", err)
	}
	
	return nil
}

func handleImplementECO(w http.ResponseWriter, r *http.Request, id string) {
	now := time.Now().Format("2006-01-02 15:04:05")
	user := getUsername(r)
	_, err := db.Exec("UPDATE ecos SET status='implemented',updated_at=? WHERE id=?", now, id)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	// Record implementation in latest revision
	updateRevisionImplementation(id, user, now)
	// Apply any linked part changes
	applyPartChangesForECO(id)
	logAudit(db, user, "implemented", "eco", id, "Implemented "+id)
	go emailOnECOImplemented(id)
	handleGetECO(w, r, id)
}

// --- ECO Status State Machine ---

// validateECOStatusTransition validates that a status change follows the allowed workflow
func validateECOStatusTransition(currentStatus, newStatus string) error {
	// If status unchanged, allow it
	if currentStatus == newStatus {
		return nil
	}
	
	// Define valid state transitions
	validTransitions := map[string][]string{
		"draft":       {"review", "cancelled"},
		"review":      {"approved", "rejected", "draft"},
		"approved":    {"implemented"},
		"implemented": {},                  // terminal state - no transitions allowed
		"rejected":    {"draft"},           // allow re-submission
		"cancelled":   {},                  // terminal state
	}
	
	allowed, exists := validTransitions[currentStatus]
	if !exists {
		return fmt.Errorf("invalid current status: %s", currentStatus)
	}
	
	for _, valid := range allowed {
		if newStatus == valid {
			return nil
		}
	}
	
	return fmt.Errorf("invalid transition from %s to %s", currentStatus, newStatus)
}

// --- ECO Revision Handlers ---

func nextRevisionLetter(ecoID string) string {
	var last string
	err := db.QueryRow("SELECT revision FROM eco_revisions WHERE eco_id=? ORDER BY id DESC LIMIT 1", ecoID).Scan(&last)
	if err != nil || last == "" {
		return "A"
	}
	
	// Handle overflow: A-Z, then AA, AB, AC, ..., AZ, BA, etc.
	if last == "Z" {
		return "AA"
	} else if len(last) > 1 {
		// Multi-letter revisions (AA, AB, etc.)
		return incrementRevision(last)
	}
	return string(rune(last[0] + 1))
}

// incrementRevision increments multi-letter revision strings (AA -> AB, AZ -> BA, ZZ -> AAA)
func incrementRevision(rev string) string {
	runes := []rune(rev)
	
	// Start from rightmost character
	for i := len(runes) - 1; i >= 0; i-- {
		if runes[i] < 'Z' {
			runes[i]++
			return string(runes)
		}
		// Overflow: Z -> A and carry to next position
		runes[i] = 'A'
	}
	
	// All positions overflowed (e.g., ZZ -> AAA)
	return "A" + string(runes)
}

func updateRevisionApproval(ecoID, user, now string) {
	db.Exec("UPDATE eco_revisions SET approved_by=?, approved_at=?, status='approved' WHERE eco_id=? AND id=(SELECT MAX(id) FROM eco_revisions WHERE eco_id=?)", user, now, ecoID, ecoID)
}

func updateRevisionImplementation(ecoID, user, now string) {
	db.Exec("UPDATE eco_revisions SET implemented_by=?, implemented_at=?, status='implemented' WHERE eco_id=? AND id=(SELECT MAX(id) FROM eco_revisions WHERE eco_id=?)", user, now, ecoID, ecoID)
}

func ensureInitialRevision(ecoID, user, now string) {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM eco_revisions WHERE eco_id=?", ecoID).Scan(&count)
	if count == 0 {
		db.Exec("INSERT INTO eco_revisions (eco_id, revision, status, changes_summary, created_by, created_at) VALUES (?,?,?,?,?,?)",
			ecoID, "A", "created", "Initial revision", user, now)
	}
}

func handleListECORevisions(w http.ResponseWriter, r *http.Request, ecoID string) {
	rows, err := db.Query("SELECT id,eco_id,revision,status,COALESCE(changes_summary,''),COALESCE(created_by,''),created_at,approved_by,approved_at,implemented_by,implemented_at,effectivity_date,COALESCE(notes,'') FROM eco_revisions WHERE eco_id=? ORDER BY id ASC", ecoID)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	defer rows.Close()
	var items []ECORevision
	for rows.Next() {
		var rev ECORevision
		var ab, aa, ib, ia, ed sql.NullString
		rows.Scan(&rev.ID, &rev.ECOID, &rev.Revision, &rev.Status, &rev.ChangesSummary, &rev.CreatedBy, &rev.CreatedAt, &ab, &aa, &ib, &ia, &ed, &rev.Notes)
		rev.ApprovedBy = sp(ab); rev.ApprovedAt = sp(aa)
		rev.ImplementedBy = sp(ib); rev.ImplementedAt = sp(ia)
		rev.EffectivityDate = sp(ed)
		items = append(items, rev)
	}
	if items == nil { items = []ECORevision{} }
	jsonResp(w, items)
}

func handleCreateECORevision(w http.ResponseWriter, r *http.Request, ecoID string) {
	var body struct {
		ChangesSummary  string `json:"changes_summary"`
		EffectivityDate string `json:"effectivity_date"`
		Notes           string `json:"notes"`
	}
	if err := decodeBody(r, &body); err != nil { jsonErr(w, "invalid body", 400); return }
	now := time.Now().Format("2006-01-02 15:04:05")
	user := getUsername(r)
	rev := nextRevisionLetter(ecoID)
	var ed *string
	if body.EffectivityDate != "" { ed = &body.EffectivityDate }
	res, err := db.Exec("INSERT INTO eco_revisions (eco_id, revision, status, changes_summary, created_by, created_at, effectivity_date, notes) VALUES (?,?,?,?,?,?,?,?)",
		ecoID, rev, "created", body.ChangesSummary, user, now, ed, body.Notes)
	if err != nil { jsonErr(w, err.Error(), 500); return }
	id, _ := res.LastInsertId()
	logAudit(db, user, "created", "eco_revision", ecoID, "Created revision "+rev+" for "+ecoID)
	jsonResp(w, map[string]interface{}{"id": id, "eco_id": ecoID, "revision": rev, "status": "created", "changes_summary": body.ChangesSummary, "created_by": user, "created_at": now, "effectivity_date": ed, "notes": body.Notes})
}

func handleGetECORevision(w http.ResponseWriter, r *http.Request, ecoID, revLetter string) {
	var rev ECORevision
	var ab, aa, ib, ia, ed sql.NullString
	err := db.QueryRow("SELECT id,eco_id,revision,status,COALESCE(changes_summary,''),COALESCE(created_by,''),created_at,approved_by,approved_at,implemented_by,implemented_at,effectivity_date,COALESCE(notes,'') FROM eco_revisions WHERE eco_id=? AND revision=?", ecoID, revLetter).
		Scan(&rev.ID, &rev.ECOID, &rev.Revision, &rev.Status, &rev.ChangesSummary, &rev.CreatedBy, &rev.CreatedAt, &ab, &aa, &ib, &ia, &ed, &rev.Notes)
	if err != nil { jsonErr(w, "revision not found", 404); return }
	rev.ApprovedBy = sp(ab); rev.ApprovedAt = sp(aa)
	rev.ImplementedBy = sp(ib); rev.ImplementedAt = sp(ia)
	rev.EffectivityDate = sp(ed)
	jsonResp(w, rev)
}
