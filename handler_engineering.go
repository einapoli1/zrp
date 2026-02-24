package main

import (
	"zrp/internal/handlers/engineering"
)

// engineeringHandler is the shared engineering handler instance.
var engineeringHandler *engineering.Handler

// getEngineeringHandler returns the engineering handler, lazily initializing if needed.
func getEngineeringHandler() *engineering.Handler {
	if engineeringHandler == nil || engineeringHandler.DB != db {
		engineeringHandler = &engineering.Handler{
			DB:       db,
			Hub:      wsHub,
			PartsDir: "./parts",
			NextIDFunc: func(prefix, table string, digits int) string {
				return nextID(prefix, table, digits)
			},
			RecordChangeJSON: func(userID, tableName, recordID, operation string, oldData, newData interface{}) (int64, error) {
				return recordChangeJSON(userID, tableName, recordID, operation, oldData, newData)
			},
			GetECOSnapshot: func(id string) (map[string]interface{}, error) {
				return getECOSnapshot(id)
			},
			GetPartByIPN: func(partsDir, ipn string) (map[string]string, error) {
				return getPartByIPN(partsDir, ipn)
			},
			EmailOnECOApproved: func(id string) {
				emailOnECOApproved(id)
			},
			EmailOnECOImplemented: func(id string) {
				emailOnECOImplemented(id)
			},
			ApplyPartChangesForECO: func(id string) error {
				return applyPartChangesForECO(id)
			},
			GetPartMPN: func(ipn string) string {
				return getPartMPN(ipn)
			},
			GetAppSetting: func(key string) string {
				return getAppSetting(key)
			},
			SetAppSetting: func(key, value string) error {
				return setAppSetting(key, value)
			},
		}
	}
	return engineeringHandler
}
