// Package healthpb implements the Health trait.
// Health in SC BOS is a managed trait, which means the SC BOS node implements the HealthApiServer so you don't have to.
// Instead, health checks are created via a [*Checks] instance and retained in a [*Registry].
// Individual health checks are updated by the caller depending on the type of check.
// For example a [BoundsCheck] may have its value updated periodically to reflect the current state of the system being monitored.
package healthpb
