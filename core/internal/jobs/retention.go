package jobs

import "time"

// Standardized data-retention windows shared by the local cleanup jobs (sessions,
// vouchers, notifications). One place to tune the policy:
//
//   - Used/consumed resources (consumed/expired sessions, activated vouchers) are
//     kept usedResourceRetentionDays after they are used, then deleted.
//   - Unused/unactivated resources (unstarted sessions, never-activated vouchers)
//     are NEVER auto-deleted; once older than unusedResourceMinAgeDays a daily
//     warning is raised for the admin instead.
//   - Notifications are deleted readNotificationRetentionDays after being read
//     (unread rows are kept, bounded only by the newest-N backstop cap).
//
// The sessions delete/count SQL hardcodes the 30-day grace inline (its per-row
// exp_days arithmetic can't be expressed as a single Go cutoff); keep that literal
// in sync with usedResourceRetentionDays.
const (
	usedResourceRetentionDays     = 30
	unusedResourceMinAgeDays      = 90
	readNotificationRetentionDays = 30
)

// unusedNotifyThrottle suppresses a repeat of an identical cleanup warning (same
// subject) within this window, so a persistent condition (unstarted sessions,
// unused vouchers) nags at most once per day rather than on every job tick. The
// prod jobs already run daily; 20h dedupes dev's fast loop and same-day retries
// without suppressing the next calendar day's warning.
const unusedNotifyThrottle = 20 * time.Hour
