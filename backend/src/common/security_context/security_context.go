// Package securitycontext provides shared security-context helpers for the backend.
package securitycontext

// IsRunAsNonRootEffective returns whether runAsNonRoot is effectively
// true given admin and component settings. Admin takes precedence;
// component value is used only when admin is nil.
func IsRunAsNonRootEffective(admin, component *bool) bool {
	if admin != nil {
		return *admin
	}
	if component != nil {
		return *component
	}
	return false
}
