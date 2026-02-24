package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// handleListConfigTemplates - GET /api/v1/configurator/templates
func handleListConfigTemplates(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT id, name, model_format, created_at, updated_at 
		FROM configuration_templates 
		ORDER BY created_at DESC
	`)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var templates []ConfigurationTemplate
	for rows.Next() {
		var t ConfigurationTemplate
		if err := rows.Scan(&t.ID, &t.Name, &t.ModelFormat, &t.CreatedAt, &t.UpdatedAt); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}

		// Get counts for parameters and parts
		var paramCount, partCount, genCount int
		db.QueryRow("SELECT COUNT(*) FROM configuration_parameters WHERE template_id=?", t.ID).Scan(&paramCount)
		db.QueryRow("SELECT COUNT(*) FROM configuration_parts WHERE template_id=?", t.ID).Scan(&partCount)
		db.QueryRow("SELECT COUNT(*) FROM configuration_generations WHERE template_id=?", t.ID).Scan(&genCount)

		templates = append(templates, t)
	}

	if templates == nil {
		templates = []ConfigurationTemplate{}
	}
	jsonResp(w, templates)
}

// handleGetConfigTemplate - GET /api/v1/configurator/templates/:id
func handleGetConfigTemplate(w http.ResponseWriter, r *http.Request, id string) {
	templateID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid template ID", 400)
		return
	}

	var t ConfigurationTemplate
	err = db.QueryRow("SELECT id, name, model_format, created_at, updated_at FROM configuration_templates WHERE id=?", templateID).
		Scan(&t.ID, &t.Name, &t.ModelFormat, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonErr(w, "template not found", 404)
		} else {
			jsonErr(w, err.Error(), 500)
		}
		return
	}

	// Load parameters
	params, err := db.Query("SELECT id, template_id, name, type, values_json, created_at FROM configuration_parameters WHERE template_id=?", templateID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer params.Close()

	var parameters []ConfigurationParameter
	for params.Next() {
		var p ConfigurationParameter
		if err := params.Scan(&p.ID, &p.TemplateID, &p.Name, &p.Type, &p.ValuesJSON, &p.CreatedAt); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		parameters = append(parameters, p)
	}
	if parameters == nil {
		parameters = []ConfigurationParameter{}
	}
	t.Parameters = parameters

	// Load parts
	parts, err := db.Query("SELECT id, template_id, ipn, quantity, include_all_variants, constraints_json, created_at FROM configuration_parts WHERE template_id=?", templateID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer parts.Close()

	var configParts []ConfigurationPart
	for parts.Next() {
		var p ConfigurationPart
		if err := parts.Scan(&p.ID, &p.TemplateID, &p.IPN, &p.Quantity, &p.IncludeAllVariants, &p.ConstraintsJSON, &p.CreatedAt); err != nil {
			jsonErr(w, err.Error(), 500)
			return
		}
		// Enrich with part description
		var desc string
		db.QueryRow("SELECT COALESCE(description,'') FROM parts WHERE ipn=?", p.IPN).Scan(&desc)
		p.Description = desc
		configParts = append(configParts, p)
	}
	if configParts == nil {
		configParts = []ConfigurationPart{}
	}
	t.Parts = configParts

	jsonResp(w, t)
}

// handleCreateConfigTemplate - POST /api/v1/configurator/templates
func handleCreateConfigTemplate(w http.ResponseWriter, r *http.Request) {
	var t ConfigurationTemplate
	if err := decodeBody(r, &t); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	requireField(ve, "name", t.Name)
	requireField(ve, "model_format", t.ModelFormat)
	validateMaxLength(ve, "name", t.Name, 255)
	validateMaxLength(ve, "model_format", t.ModelFormat, 255)

	// Validate model_format contains at least one {param}
	if !strings.Contains(t.ModelFormat, "{") || !strings.Contains(t.ModelFormat, "}") {
		ve.Add("model_format", "must contain at least one {param} placeholder")
	}

	if ve.HasErrors() {
		jsonErr(w, ve.Error(), 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	result, err := db.Exec("INSERT INTO configuration_templates (name, model_format, created_at, updated_at) VALUES (?, ?, ?, ?)",
		t.Name, t.ModelFormat, now, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	id, _ := result.LastInsertId()
	t.ID = int(id)
	t.CreatedAt = now
	t.UpdatedAt = now
	t.Parameters = []ConfigurationParameter{}
	t.Parts = []ConfigurationPart{}

	logAudit(db, getUsername(r), "created", "configuration_template", fmt.Sprintf("%d", t.ID), "Created template: "+t.Name)
	jsonResp(w, t)
}

// handleUpdateConfigTemplate - PUT /api/v1/configurator/templates/:id
func handleUpdateConfigTemplate(w http.ResponseWriter, r *http.Request, id string) {
	templateID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid template ID", 400)
		return
	}

	// Verify exists
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM configuration_templates WHERE id=?", templateID).Scan(&exists)
	if exists == 0 {
		jsonErr(w, "template not found", 404)
		return
	}

	var t ConfigurationTemplate
	if err := decodeBody(r, &t); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	requireField(ve, "name", t.Name)
	requireField(ve, "model_format", t.ModelFormat)
	validateMaxLength(ve, "name", t.Name, 255)
	validateMaxLength(ve, "model_format", t.ModelFormat, 255)

	// Validate model_format contains at least one {param}
	if !strings.Contains(t.ModelFormat, "{") || !strings.Contains(t.ModelFormat, "}") {
		ve.Add("model_format", "must contain at least one {param} placeholder")
	}

	if ve.HasErrors() {
		jsonErr(w, ve.Error(), 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.Exec("UPDATE configuration_templates SET name=?, model_format=?, updated_at=? WHERE id=?",
		t.Name, t.ModelFormat, now, templateID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "updated", "configuration_template", fmt.Sprintf("%d", templateID), "Updated template: "+t.Name)
	handleGetConfigTemplate(w, r, id)
}

// handleDeleteConfigTemplate - DELETE /api/v1/configurator/templates/:id
func handleDeleteConfigTemplate(w http.ResponseWriter, r *http.Request, id string) {
	templateID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid template ID", 400)
		return
	}

	var name string
	err = db.QueryRow("SELECT name FROM configuration_templates WHERE id=?", templateID).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonErr(w, "template not found", 404)
		} else {
			jsonErr(w, err.Error(), 500)
		}
		return
	}

	_, err = db.Exec("DELETE FROM configuration_templates WHERE id=?", templateID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "deleted", "configuration_template", fmt.Sprintf("%d", templateID), "Deleted template: "+name)
	jsonResp(w, map[string]string{"status": "deleted"})
}

// Parameter handlers

// handleCreateConfigParameter - POST /api/v1/configurator/templates/:id/parameters
func handleCreateConfigParameter(w http.ResponseWriter, r *http.Request, id string) {
	templateID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid template ID", 400)
		return
	}

	// Verify template exists
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM configuration_templates WHERE id=?", templateID).Scan(&exists)
	if exists == 0 {
		jsonErr(w, "template not found", 404)
		return
	}

	var p ConfigurationParameter
	if err := decodeBody(r, &p); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	requireField(ve, "name", p.Name)
	requireField(ve, "type", p.Type)
	requireField(ve, "values_json", p.ValuesJSON)
	validateEnum(ve, "type", p.Type, []string{"enum", "range"})
	validateMaxLength(ve, "name", p.Name, 100)

	// Validate parameter name is alphanumeric + underscore
	if !isValidParameterName(p.Name) {
		ve.Add("name", "must be alphanumeric with underscores only")
	}

	// Validate JSON structure based on type
	if p.Type == "enum" {
		var values []string
		if err := json.Unmarshal([]byte(p.ValuesJSON), &values); err != nil {
			ve.Add("values_json", "must be a JSON array for enum type")
		}
	} else if p.Type == "range" {
		var rangeVal map[string]interface{}
		if err := json.Unmarshal([]byte(p.ValuesJSON), &rangeVal); err != nil {
			ve.Add("values_json", "must be a JSON object for range type")
		} else {
			if _, ok := rangeVal["min"]; !ok {
				ve.Add("values_json", "range must have 'min' field")
			}
			if _, ok := rangeVal["max"]; !ok {
				ve.Add("values_json", "range must have 'max' field")
			}
		}
	}

	if ve.HasErrors() {
		jsonErr(w, ve.Error(), 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	result, err := db.Exec("INSERT INTO configuration_parameters (template_id, name, type, values_json, created_at) VALUES (?, ?, ?, ?, ?)",
		templateID, p.Name, p.Type, p.ValuesJSON, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	paramID, _ := result.LastInsertId()
	p.ID = int(paramID)
	p.TemplateID = templateID
	p.CreatedAt = now

	logAudit(db, getUsername(r), "created", "configuration_parameter", fmt.Sprintf("%d", p.ID), "Added parameter: "+p.Name)
	jsonResp(w, p)
}

// handleUpdateConfigParameter - PUT /api/v1/configurator/parameters/:id
func handleUpdateConfigParameter(w http.ResponseWriter, r *http.Request, id string) {
	paramID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid parameter ID", 400)
		return
	}

	// Verify exists
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM configuration_parameters WHERE id=?", paramID).Scan(&exists)
	if exists == 0 {
		jsonErr(w, "parameter not found", 404)
		return
	}

	var p ConfigurationParameter
	if err := decodeBody(r, &p); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	requireField(ve, "name", p.Name)
	requireField(ve, "type", p.Type)
	requireField(ve, "values_json", p.ValuesJSON)
	validateEnum(ve, "type", p.Type, []string{"enum", "range"})
	validateMaxLength(ve, "name", p.Name, 100)

	// Validate parameter name is alphanumeric + underscore
	if !isValidParameterName(p.Name) {
		ve.Add("name", "must be alphanumeric with underscores only")
	}

	// Validate JSON structure based on type
	if p.Type == "enum" {
		var values []string
		if err := json.Unmarshal([]byte(p.ValuesJSON), &values); err != nil {
			ve.Add("values_json", "must be a JSON array for enum type")
		}
	} else if p.Type == "range" {
		var rangeVal map[string]interface{}
		if err := json.Unmarshal([]byte(p.ValuesJSON), &rangeVal); err != nil {
			ve.Add("values_json", "must be a JSON object for range type")
		} else {
			if _, ok := rangeVal["min"]; !ok {
				ve.Add("values_json", "range must have 'min' field")
			}
			if _, ok := rangeVal["max"]; !ok {
				ve.Add("values_json", "range must have 'max' field")
			}
		}
	}

	if ve.HasErrors() {
		jsonErr(w, ve.Error(), 400)
		return
	}

	_, err = db.Exec("UPDATE configuration_parameters SET name=?, type=?, values_json=? WHERE id=?",
		p.Name, p.Type, p.ValuesJSON, paramID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "updated", "configuration_parameter", fmt.Sprintf("%d", paramID), "Updated parameter: "+p.Name)
	
	// Return updated parameter
	var updated ConfigurationParameter
	db.QueryRow("SELECT id, template_id, name, type, values_json, created_at FROM configuration_parameters WHERE id=?", paramID).
		Scan(&updated.ID, &updated.TemplateID, &updated.Name, &updated.Type, &updated.ValuesJSON, &updated.CreatedAt)
	jsonResp(w, updated)
}

// handleDeleteConfigParameter - DELETE /api/v1/configurator/parameters/:id
func handleDeleteConfigParameter(w http.ResponseWriter, r *http.Request, id string) {
	paramID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid parameter ID", 400)
		return
	}

	var name string
	err = db.QueryRow("SELECT name FROM configuration_parameters WHERE id=?", paramID).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonErr(w, "parameter not found", 404)
		} else {
			jsonErr(w, err.Error(), 500)
		}
		return
	}

	_, err = db.Exec("DELETE FROM configuration_parameters WHERE id=?", paramID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "deleted", "configuration_parameter", fmt.Sprintf("%d", paramID), "Deleted parameter: "+name)
	jsonResp(w, map[string]string{"status": "deleted"})
}

// Part handlers

// handleCreateConfigPart - POST /api/v1/configurator/templates/:id/parts
func handleCreateConfigPart(w http.ResponseWriter, r *http.Request, id string) {
	templateID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid template ID", 400)
		return
	}

	// Verify template exists
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM configuration_templates WHERE id=?", templateID).Scan(&exists)
	if exists == 0 {
		jsonErr(w, "template not found", 404)
		return
	}

	var p ConfigurationPart
	if err := decodeBody(r, &p); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	requireField(ve, "ipn", p.IPN)
	if p.Quantity <= 0 {
		ve.Add("quantity", "must be greater than 0")
	}

	// Verify IPN exists in parts table
	var partExists int
	db.QueryRow("SELECT COUNT(*) FROM parts WHERE ipn=?", p.IPN).Scan(&partExists)
	if partExists == 0 {
		ve.Add("ipn", "part not found")
	}

	// Validate constraints JSON
	if p.ConstraintsJSON != "" {
		var constraints map[string]interface{}
		if err := json.Unmarshal([]byte(p.ConstraintsJSON), &constraints); err != nil {
			ve.Add("constraints_json", "must be valid JSON object")
		}
	} else {
		p.ConstraintsJSON = "{}"
	}

	if ve.HasErrors() {
		jsonErr(w, ve.Error(), 400)
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	result, err := db.Exec("INSERT INTO configuration_parts (template_id, ipn, quantity, include_all_variants, constraints_json, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		templateID, p.IPN, p.Quantity, p.IncludeAllVariants, p.ConstraintsJSON, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	partID, _ := result.LastInsertId()
	p.ID = int(partID)
	p.TemplateID = templateID
	p.CreatedAt = now

	// Enrich with description
	var desc string
	db.QueryRow("SELECT COALESCE(description,'') FROM parts WHERE ipn=?", p.IPN).Scan(&desc)
	p.Description = desc

	logAudit(db, getUsername(r), "created", "configuration_part", fmt.Sprintf("%d", p.ID), "Added part: "+p.IPN)
	jsonResp(w, p)
}

// handleUpdateConfigPart - PUT /api/v1/configurator/parts/:id
func handleUpdateConfigPart(w http.ResponseWriter, r *http.Request, id string) {
	partID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid part ID", 400)
		return
	}

	// Verify exists
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM configuration_parts WHERE id=?", partID).Scan(&exists)
	if exists == 0 {
		jsonErr(w, "part not found", 404)
		return
	}

	var p ConfigurationPart
	if err := decodeBody(r, &p); err != nil {
		jsonErr(w, "invalid body", 400)
		return
	}

	ve := &ValidationErrors{}
	if p.Quantity <= 0 {
		ve.Add("quantity", "must be greater than 0")
	}

	// Validate constraints JSON
	if p.ConstraintsJSON != "" {
		var constraints map[string]interface{}
		if err := json.Unmarshal([]byte(p.ConstraintsJSON), &constraints); err != nil {
			ve.Add("constraints_json", "must be valid JSON object")
		}
	}

	if ve.HasErrors() {
		jsonErr(w, ve.Error(), 400)
		return
	}

	_, err = db.Exec("UPDATE configuration_parts SET quantity=?, include_all_variants=?, constraints_json=? WHERE id=?",
		p.Quantity, p.IncludeAllVariants, p.ConstraintsJSON, partID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "updated", "configuration_part", fmt.Sprintf("%d", partID), "Updated part config")
	
	// Return updated part
	var updated ConfigurationPart
	db.QueryRow("SELECT id, template_id, ipn, quantity, include_all_variants, constraints_json, created_at FROM configuration_parts WHERE id=?", partID).
		Scan(&updated.ID, &updated.TemplateID, &updated.IPN, &updated.Quantity, &updated.IncludeAllVariants, &updated.ConstraintsJSON, &updated.CreatedAt)
	
	// Enrich with description
	var desc string
	db.QueryRow("SELECT COALESCE(description,'') FROM parts WHERE ipn=?", updated.IPN).Scan(&desc)
	updated.Description = desc

	jsonResp(w, updated)
}

// handleDeleteConfigPart - DELETE /api/v1/configurator/parts/:id
func handleDeleteConfigPart(w http.ResponseWriter, r *http.Request, id string) {
	partID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid part ID", 400)
		return
	}

	var ipn string
	err = db.QueryRow("SELECT ipn FROM configuration_parts WHERE id=?", partID).Scan(&ipn)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonErr(w, "part not found", 404)
		} else {
			jsonErr(w, err.Error(), 500)
		}
		return
	}

	_, err = db.Exec("DELETE FROM configuration_parts WHERE id=?", partID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "deleted", "configuration_part", fmt.Sprintf("%d", partID), "Removed part: "+ipn)
	jsonResp(w, map[string]string{"status": "deleted"})
}

// Generation handlers

// handlePreviewConfigVariants - GET /api/v1/configurator/templates/:id/preview
func handlePreviewConfigVariants(w http.ResponseWriter, r *http.Request, id string) {
	templateID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid template ID", 400)
		return
	}

	variants, err := generateVariants(templateID, 10)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	jsonResp(w, map[string]interface{}{
		"preview":       variants,
		"total_count":   len(variants),
		"showing_first": 10,
	})
}

// handleGenerateConfigVariants - POST /api/v1/configurator/templates/:id/generate
func handleGenerateConfigVariants(w http.ResponseWriter, r *http.Request, id string) {
	templateID, err := strconv.Atoi(id)
	if err != nil {
		jsonErr(w, "invalid template ID", 400)
		return
	}

	// Get template name
	var templateName string
	err = db.QueryRow("SELECT name FROM configuration_templates WHERE id=?", templateID).Scan(&templateName)
	if err != nil {
		if err == sql.ErrNoRows {
			jsonErr(w, "template not found", 404)
		} else {
			jsonErr(w, err.Error(), 500)
		}
		return
	}

	// Generate all variants (no limit)
	variants, err := generateVariants(templateID, 0)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	if len(variants) == 0 {
		jsonErr(w, "no variants generated - check template configuration", 400)
		return
	}

	// Create ECO
	ecoID := nextID("ECO", "ecos", 3)
	ecoTitle := fmt.Sprintf("Configuration: %s - %d variants", templateName, len(variants))
	now := time.Now().Format("2006-01-02 15:04:05")

	tx, err := db.Begin()
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	// Create ECO with type "configuration"
	_, err = tx.Exec("INSERT INTO ecos (id, title, description, status, priority, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		ecoID, ecoTitle, fmt.Sprintf("Auto-generated from configurator template: %s", templateName), "pending", "normal", now, now)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Add all variants as proposed parts in the ECO
	// (Note: This assumes eco_proposed_parts table exists or we use a similar mechanism)
	// For now, we'll add them to the affected_ipns field as JSON
	ipnList := make([]string, len(variants))
	for i, v := range variants {
		ipnList[i] = v["ipn"].(string)
	}
	ipnsJSON, _ := json.Marshal(ipnList)
	_, err = tx.Exec("UPDATE ecos SET affected_ipns=? WHERE id=?", string(ipnsJSON), ecoID)
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	// Record generation
	_, err = tx.Exec("INSERT INTO configuration_generations (template_id, eco_id, generated_at, variant_count) VALUES (?, ?, ?, ?)",
		templateID, ecoID, now, len(variants))
	if err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	if err = tx.Commit(); err != nil {
		jsonErr(w, err.Error(), 500)
		return
	}

	logAudit(db, getUsername(r), "generated", "configuration", ecoID, fmt.Sprintf("Generated %d variants from template %s", len(variants), templateName))

	// Return first 10 IPNs for preview
	preview := ipnList
	if len(preview) > 10 {
		preview = preview[:10]
	}

	jsonResp(w, map[string]interface{}{
		"eco_id":        ecoID,
		"variant_count": len(variants),
		"preview":       preview,
	})
}

// Helper functions

func isValidParameterName(name string) bool {
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return len(name) > 0
}

// generateVariants creates all possible configurations
// limit=0 means generate all, limit>0 generates first N
func generateVariants(templateID int, limit int) ([]map[string]interface{}, error) {
	// Get template
	var modelFormat string
	err := db.QueryRow("SELECT model_format FROM configuration_templates WHERE id=?", templateID).Scan(&modelFormat)
	if err != nil {
		return nil, err
	}

	// Get parameters
	params, err := db.Query("SELECT name, type, values_json FROM configuration_parameters WHERE template_id=?", templateID)
	if err != nil {
		return nil, err
	}
	defer params.Close()

	var parameters []map[string]interface{}
	for params.Next() {
		var name, ptype, valuesJSON string
		if err := params.Scan(&name, &ptype, &valuesJSON); err != nil {
			return nil, err
		}
		parameters = append(parameters, map[string]interface{}{
			"name":        name,
			"type":        ptype,
			"values_json": valuesJSON,
		})
	}

	if len(parameters) == 0 {
		return nil, fmt.Errorf("template has no parameters")
	}

	// Get parts
	parts, err := db.Query("SELECT ipn, quantity, include_all_variants, constraints_json FROM configuration_parts WHERE template_id=?", templateID)
	if err != nil {
		return nil, err
	}
	defer parts.Close()

	var configParts []map[string]interface{}
	for parts.Next() {
		var ipn, constraintsJSON string
		var quantity, includeAll int
		if err := parts.Scan(&ipn, &quantity, &includeAll, &constraintsJSON); err != nil {
			return nil, err
		}
		configParts = append(configParts, map[string]interface{}{
			"ipn":                  ipn,
			"quantity":             quantity,
			"include_all_variants": includeAll,
			"constraints_json":     constraintsJSON,
		})
	}

	// Generate all combinations
	combinations := generateCombinations(parameters)
	var variants []map[string]interface{}

	for _, combo := range combinations {
		if limit > 0 && len(variants) >= limit {
			break
		}

		// Generate IPN
		ipn := modelFormat
		for paramName, paramValue := range combo {
			placeholder := "{" + paramName + "}"
			ipn = strings.ReplaceAll(ipn, placeholder, fmt.Sprintf("%v", paramValue))
		}

		// Build BOM
		bom := []map[string]interface{}{}
		for _, part := range configParts {
			partIPN := part["ipn"].(string)
			quantity := part["quantity"].(int)
			includeAll := part["include_all_variants"].(int)
			constraintsJSON := part["constraints_json"].(string)

			if includeAll == 1 {
				bom = append(bom, map[string]interface{}{
					"ipn":      partIPN,
					"quantity": quantity,
				})
			} else if matchesConstraints(constraintsJSON, combo) {
				bom = append(bom, map[string]interface{}{
					"ipn":      partIPN,
					"quantity": quantity,
				})
			}
		}

		variants = append(variants, map[string]interface{}{
			"ipn":       ipn,
			"bom_count": len(bom),
			"bom":       bom,
		})
	}

	return variants, nil
}

func generateCombinations(parameters []map[string]interface{}) []map[string]interface{} {
	if len(parameters) == 0 {
		return []map[string]interface{}{}
	}

	// Recursive combination generator
	var results []map[string]interface{}
	var recurse func(int, map[string]interface{})
	
	recurse = func(index int, current map[string]interface{}) {
		if index >= len(parameters) {
			// Make a copy
			combo := make(map[string]interface{})
			for k, v := range current {
				combo[k] = v
			}
			results = append(results, combo)
			return
		}

		param := parameters[index]
		name := param["name"].(string)
		ptype := param["type"].(string)
		valuesJSON := param["values_json"].(string)

		if ptype == "enum" {
			var values []string
			json.Unmarshal([]byte(valuesJSON), &values)
			for _, val := range values {
				current[name] = val
				recurse(index+1, current)
			}
		} else if ptype == "range" {
			var rangeVal map[string]interface{}
			json.Unmarshal([]byte(valuesJSON), &rangeVal)
			// For range, we'll just use min and max as discrete values
			// In a real implementation, you might want to sample the range
			min := rangeVal["min"]
			max := rangeVal["max"]
			
			current[name] = min
			recurse(index+1, current)
			if min != max {
				current[name] = max
				recurse(index+1, current)
			}
		}
	}

	recurse(0, make(map[string]interface{}))
	return results
}

func matchesConstraints(constraintsJSON string, params map[string]interface{}) bool {
	if constraintsJSON == "" || constraintsJSON == "{}" {
		return true
	}

	var constraints map[string]interface{}
	if err := json.Unmarshal([]byte(constraintsJSON), &constraints); err != nil {
		return false
	}

	for paramName, constraint := range constraints {
		paramValue, exists := params[paramName]
		if !exists {
			return false
		}

		// Check if constraint is an array (enum) or object (range)
		switch c := constraint.(type) {
		case []interface{}:
			// Enum constraint
			found := false
			for _, allowed := range c {
				if fmt.Sprintf("%v", paramValue) == fmt.Sprintf("%v", allowed) {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		case map[string]interface{}:
			// Range constraint
			min, hasMin := c["min"]
			max, hasMax := c["max"]
			if hasMin || hasMax {
				// Try to convert to float for comparison
				var val float64
				switch v := paramValue.(type) {
				case float64:
					val = v
				case int:
					val = float64(v)
				case string:
					// Try parsing numeric value from string
					fmt.Sscanf(v, "%f", &val)
				}

				if hasMin {
					var minVal float64
					switch m := min.(type) {
					case float64:
						minVal = m
					case int:
						minVal = float64(m)
					}
					if val < minVal {
						return false
					}
				}
				if hasMax {
					var maxVal float64
					switch m := max.(type) {
					case float64:
						maxVal = m
					case int:
						maxVal = float64(m)
					}
					if val > maxVal {
						return false
					}
				}
			}
		}
	}

	return true
}
