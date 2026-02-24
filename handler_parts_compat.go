package main

import (
	"net/http"
	"zrp/internal/handlers/parts"
	"zrp/internal/models"
)

// partsHandler is the shared parts handler instance.
var partsHandler *parts.Handler

// getPartsHandler returns the parts handler, lazily initializing if needed.
func getPartsHandler() *parts.Handler {
	if partsHandler == nil || partsHandler.DB != db {
		partsHandler = &parts.Handler{
			DB:       db,
			Hub:      wsHub,
			PartsDir: "./parts",
			NextID: func(prefix, table string, digits int) string {
				return nextID(prefix, table, digits)
			},
			EnsureInitialRevision: func(ecoID, user, now string) {
				ensureInitialRevision(ecoID, user, now)
			},
			SnapshotDocumentVersion: func(docID, changeSummary, createdBy string, ecoID *string) error {
				return snapshotDocumentVersion(docID, changeSummary, createdBy, ecoID)
			},
			HandleGetDoc: func(w http.ResponseWriter, r *http.Request, id string) {
				handleGetDoc(w, r, id)
			},
			LoadPartsFromDir: func() (map[string][]models.Part, map[string][]string, map[string]string, error) {
				cats, catOrder, catDesc, err := loadPartsFromDir()
				// Convert []Part to []models.Part
				result := make(map[string][]models.Part)
				for k, parts := range cats {
					mparts := make([]models.Part, len(parts))
					for i, p := range parts {
						mparts[i] = models.Part{IPN: p.IPN, Fields: p.Fields}
					}
					result[k] = mparts
				}
				return result, catOrder, catDesc, err
			},
			GetPartByIPN: func(partsDir, ipn string) (map[string]string, error) {
				return getPartByIPN(partsDir, ipn)
			},
			LogSensitiveDataAccess: func(r *http.Request, dataType, recordID, details string) {
				// TODO: Implement sensitive data access logging if needed
			},
		}
	}
	return partsHandler
}
