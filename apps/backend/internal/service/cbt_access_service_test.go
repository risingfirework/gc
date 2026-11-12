package service

import (
	"testing"
	"time"

	"tka/apps/backend/internal/domain"
)

func TestResolveCBTAccess(t *testing.T) {
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	past := now.AddDate(0, 0, -1)
	future := now.AddDate(0, 0, 30)

	base := domain.CBTAccessSettings{
		AccessMode:             domain.CBTAccessModeTrialThenPaid,
		FreeQuota:              1,
		GraceDays:              7,
		RequireVerifiedTeacher: true,
	}

	tests := []struct {
		name        string
		settings    domain.CBTAccessSettings
		entitlement domain.TeacherCBTEntitlement
		verified    string
		pending     *domain.TeacherCBTPayment
		wantCan     bool
		wantReason  string
	}{
		{
			name:        "unverified teacher is blocked",
			settings:    base,
			entitlement: domain.TeacherCBTEntitlement{Status: domain.CBTAccessEntitlementTrialAvailable},
			verified:    domain.TeacherVerificationPending,
			wantCan:     false,
			wantReason:  domain.CBTAccessReasonVerifiedRequired,
		},
		{
			name:        "free mode always allows verified teacher",
			settings:    domain.CBTAccessSettings{AccessMode: domain.CBTAccessModeFree, RequireVerifiedTeacher: true},
			entitlement: domain.TeacherCBTEntitlement{Status: domain.CBTAccessEntitlementExpired},
			verified:    domain.TeacherVerificationApproved,
			wantCan:     true,
			wantReason:  domain.CBTAccessReasonFreeMode,
		},
		{
			name:        "trial quota available",
			settings:    base,
			entitlement: domain.TeacherCBTEntitlement{Status: domain.CBTAccessEntitlementTrialAvailable, FreeQuotaUsed: 0},
			verified:    domain.TeacherVerificationApproved,
			wantCan:     true,
			wantReason:  domain.CBTAccessReasonTrialAvailable,
		},
		{
			name:        "paid access is active",
			settings:    base,
			entitlement: domain.TeacherCBTEntitlement{Status: domain.CBTAccessEntitlementActive, FreeQuotaUsed: 1, ActiveUntil: &future},
			verified:    domain.TeacherVerificationApproved,
			wantCan:     true,
			wantReason:  domain.CBTAccessReasonActive,
		},
		{
			name:        "expired access within grace period still allowed",
			settings:    base,
			entitlement: domain.TeacherCBTEntitlement{Status: domain.CBTAccessEntitlementExpired, FreeQuotaUsed: 1, ActiveUntil: &past},
			verified:    domain.TeacherVerificationApproved,
			wantCan:     true,
			wantReason:  domain.CBTAccessReasonGracePeriod,
		},
		{
			name:        "expired access beyond grace requires payment",
			settings:    func() domain.CBTAccessSettings { s := base; s.GraceDays = 0; return s }(),
			entitlement: domain.TeacherCBTEntitlement{Status: domain.CBTAccessEntitlementExpired, FreeQuotaUsed: 1, ActiveUntil: &past},
			verified:    domain.TeacherVerificationApproved,
			wantCan:     false,
			wantReason:  domain.CBTAccessReasonExpired,
		},
		{
			name:        "pending payment blocks new creation",
			settings:    func() domain.CBTAccessSettings { s := base; s.GraceDays = 0; return s }(),
			entitlement: domain.TeacherCBTEntitlement{Status: domain.CBTAccessEntitlementTrialUsed, FreeQuotaUsed: 1},
			verified:    domain.TeacherVerificationApproved,
			pending:     &domain.TeacherCBTPayment{Status: domain.CBTPaymentPending},
			wantCan:     false,
			wantReason:  domain.CBTAccessReasonPendingReview,
		},
		{
			name:        "blocked access is denied even with trial quota",
			settings:    base,
			entitlement: domain.TeacherCBTEntitlement{Status: domain.CBTAccessEntitlementBlocked, FreeQuotaUsed: 0},
			verified:    domain.TeacherVerificationApproved,
			wantCan:     false,
			wantReason:  domain.CBTAccessReasonAccessDenied,
		},
		{
			name:        "paid mode ignores trial quota",
			settings:    func() domain.CBTAccessSettings { s := base; s.AccessMode = domain.CBTAccessModePaid; return s }(),
			entitlement: domain.TeacherCBTEntitlement{Status: domain.CBTAccessEntitlementTrialAvailable, FreeQuotaUsed: 0},
			verified:    domain.TeacherVerificationApproved,
			wantCan:     false,
			wantReason:  domain.CBTAccessReasonPaymentRequired,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			can, reason, _ := resolveCBTAccess(&test.settings, &test.entitlement, test.verified, test.pending, now)
			if can != test.wantCan || reason != test.wantReason {
				t.Fatalf("got can=%v reason=%q, want can=%v reason=%q", can, reason, test.wantCan, test.wantReason)
			}
		})
	}
}
