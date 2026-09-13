import axios, { AxiosError } from "axios";

const TOKEN_KEY = "tka_access_token";
const LEGACY_TOKEN_KEY = "tka_token";
const USER_KEY = "tka_current_user";
const REFRESH_KEY = "tka_refresh_token";
export const SESSION_INVALID_EVENT = "tka:session-invalid";

declare module "axios" {
  export interface InternalAxiosRequestConfig {
    _retried?: boolean;
  }
}

let refreshPromise: Promise<boolean> | null = null;

async function tryRefreshSession(): Promise<boolean> {
  const refreshToken = typeof window === "undefined" ? null : localStorage.getItem(REFRESH_KEY);
  if (!refreshToken) return false;
  try {
    const response = (await axios.post<LoginResponse>(`${http.defaults.baseURL}/auth/refresh`, { refresh_token: refreshToken }, { headers: { "Content-Type": "application/json", Accept: "application/json" }, timeout: 10_000 })).data;
    if (!response.access_token) return false;
    tokenStore.set(response.access_token, response.user);
    if (response.refresh_token) tokenStore.setRefresh(response.refresh_token);
    return true;
  } catch {
    return false;
  }
}

function expireSession(message: string) {
  tokenStore.clear();
  window.dispatchEvent(new CustomEvent(SESSION_INVALID_EVENT, { detail: message }));
  if (!window.location.pathname.endsWith("/login")) window.location.replace("/login?reason=session-expired");
}

export class APIError extends Error {
  constructor(message: string, public status?: number, public code?: string, public networkError = false) {
    super(message);
    this.name = "APIError";
  }
}

export const http = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080/api/v1",
  timeout: 10_000,
  headers: { "Content-Type": "application/json", Accept: "application/json" },
});

http.interceptors.request.use((config) => {
  if (typeof window !== "undefined") {
    const token = localStorage.getItem(TOKEN_KEY) ?? localStorage.getItem(LEGACY_TOKEN_KEY);
    if (token) config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => response,
  (reason: AxiosError<{ error?: string; message?: string; code?: string }>) => {
    const status = reason.response?.status;
    const body = reason.response?.data;
    const networkError = !reason.response;
    const message = body?.message ?? body?.error ?? (networkError ? "Tidak dapat terhubung ke server." : "Permintaan gagal.");
    const requestPath = reason.config?.url ?? "";
    const isPreAuthRequest = ["/auth/login", "/auth/register", "/auth/google", "/auth/verify-2fa"].some((path) => requestPath.endsWith(path));
    const isRefreshRequest = requestPath.endsWith("/auth/refresh");
    const originalRequest = reason.config;
    if (status === 401 && !isPreAuthRequest && !isRefreshRequest && originalRequest && typeof window !== "undefined") {
      if (!originalRequest._retried) {
        originalRequest._retried = true;
        refreshPromise ??= tryRefreshSession().finally(() => { refreshPromise = null; });
        return refreshPromise.then((ok) => {
          if (!ok) {
            expireSession(message);
            throw new APIError(message, status, body?.code, networkError);
          }
          return http.request(originalRequest);
        });
      }
      if (!refreshPromise) expireSession(message);
    }
    return Promise.reject(new APIError(message, status, body?.code, networkError));
  },
);

export type User = { id: string; email: string; name: string; birth_date: string; phone: string; role: "student" | "teacher" | "admin" | "owner" | "finance" | "affiliate"; referral_code?: string; school_level: "SD" | "SMP" | "SMA"; totp_enabled?: boolean; teacher_ktp?: string; teacher_verification_status?: "pending" | "approved" | "rejected"; teacher_rejection_reason?: string; teacher_appeal_image?: string; created_at: string; updated_at: string };
export type Status = "active" | "inactive";
export type LoginResponse = { access_token?: string; token_type?: "Bearer"; expires_in?: number; refresh_token?: string; refresh_expires_in?: number; user: User; requires_2fa?: boolean; mfa_token?: string };
export type QuestionOption = { key: string; content: string; image_url?: string };
export type QuestionType = "single_choice" | "multiple_choice" | "category" | "essay";
export type PresentationType = "single" | "group";
export type TKAQuestionFields = { question_type: QuestionType; presentation_type: PresentationType; group_code: string; stimulus_text: string; question_image_url: string; stimulus_image_url: string; category_labels: string[] };
export type ExamQuestion = { id: string; subject_name: string; content_text: string; options: QuestionOption[] } & TKAQuestionFields;
export type ExamStartResponse = { user_exam_id: string; exam_id: string; title: string; status: "ongoing"; server_time: string; started_at: string; ends_at: string; remaining_seconds: number; duration_minutes: number; remaining_tokens?: number; questions: ExamQuestion[] };
export type SyncAnswerResponse = { saved: boolean; synced_at: string; remaining_seconds: number };
export type SubmitExamResponse = { user_exam_id: string; exam_id: string; status: "submitted"; finished_at: string; total_score: number; passing_score: number; passed: boolean };
export type Package = { id: string; title: string; kode: string; description: string; price: number; validity_days: number; status: Status; jenjang: string; publisher_email?: string; sales_count:number; view_count:number; exam_count:number; question_count:number; created_at: string };
export type HeroSlide = { id:string; image_data_url:string; title:string };
export type YouTubeVideo = { id:string; title:string; url:string };
export type SiteSettings = { platform_name:string; platform_tagline:string; logo_data_url:string; favicon_data_url:string; support_email:string; whatsapp:string; instagram_url:string; youtube_url:string; youtube_videos:YouTubeVideo[]; hero_interval_ms:number; catalog_interval_ms:number; default_package_validity_days:number; default_exam_duration_minutes:number; default_exam_total_questions:number; default_passing_score:number; hero_slides:HeroSlide[]; updated_at?:string };
export type OwnedPackage = Package & { expired_at: string; paid_at: string };
export type PackageExam = {
  id: string;
  package_id: string;
  title: string;
  duration_minutes: number;
  total_questions: number;
  passing_score: number;
  scoring_method: string;
  user_exam_id?: string;
  total_score?: number;
  finished_at?: string;
};
export type Transaction = { id: string; user_id: string; package_id: string; invoice_number: string; amount: number; payment_status: "pending" | "paid" | "failed" | "expired" | "refunded"; payment_method?: string; payment_url?: string; expires_at?: string; paid_at?: string; created_at: string };
export type AppNotification = { id: string; user_id: string; title: string; body: string; link?: string; read_at?: string; created_at: string };
export type MyTransaction = { id: string; invoice_number: string; package_title: string; amount: number; status: string; created_at: string };
export type PendingTransaction = { id: string; invoice_number: string; package_title: string; amount: number; payment_method?: string; payment_url: string; expires_at?: string };
export type CheckoutResponse = { transaction: Transaction; payment_url: string; expires_at: string };
export type UserPricingPolicy = { discount_percent: number; account_active: boolean };
export type SubjectResult = { subject_name: string; correct_answers: number; wrong_answers: number; unanswered: number; total_questions: number; score: number };
export type AnswerReview = { question_id: string; subject_name: string; content_text: string; question_type: "single_choice" | "multiple_choice" | "category" | "essay"; presentation_type: "single" | "group"; group_code?: string; stimulus_text?: string; question_image_url?: string; stimulus_image_url?: string; category_labels?: string[]; options: QuestionOption[]; selected_option?: string; correct_answer: string; is_correct: boolean; score_weight: number; explanation_text: string; explanation_video_url?: string };
export type ExamResult = { user_exam_id: string; exam_id: string; title: string; status: "submitted"; scoring_method: "standard" | "irt_2pl"; started_at: string; finished_at: string; total_score: number; passing_score: number; passed: boolean; correct_answers: number; wrong_answers: number; unanswered: number; subjects: SubjectResult[]; review: AnswerReview[] };
export type GlobalRanking = { rank: number; display_name: string; school_level: User["school_level"]; score: number; best_percentile: number; duration_seconds: number; finished_at: string; is_current_user: boolean; exams_done: number };
export type AdminOverview = { total_users: number; total_students: number; total_packages: number; total_exams: number; total_transactions: number; paid_transactions: number };
export type FinanceSettings = { platform_commission_percent:number; default_discount_percent:number; tax_percent:number; minimum_payout:number; payout_cycle:"weekly"|"monthly"|"manual"; auto_payout:boolean; teacher_upload_fee:number; teacher_sales_bonus_percent:number; affiliate_rate_percent:number; commission_hold_days:number; updated_at?:string };
export type UserFinanceProfile = { user_id:string; email:string; name:string; role:User["role"]; commission_percent?:number|null; discount_percent?:number|null; effective_commission_percent:number; effective_discount_percent:number; account_status:"active"|"hold"; notes:string; total_spend:number; gross_revenue:number; transaction_count:number; updated_at:string };
export type FinanceOverview = { gross_revenue:number; pending_revenue:number; platform_commission:number; estimated_payouts:number; paid_transactions:number };
export type TeacherCommission = { id:string; teacher_id:string; teacher_email:string; package_id:string; package_title:string; invoice_number?:string; kind:"upload_fee"|"sales_bonus"|"refund_reversal"|"referral_bonus"|"referral_reversal"; base_amount:number; rate_percent:number; amount:number; status:"pending"|"available"|"paid"|"cancelled"; available_at:string; paid_at?:string; payout_reference?:string; created_at:string };
export type TeacherPayout = { id:string; teacher_id:string; teacher_email:string; amount:number; reference:string; paid_at:string };
export type TeacherPayoutAccount = { teacher_id:string; method:"bank_transfer"|"e_wallet"|""; provider:string; account_number:string; account_holder_name:string; phone:string; updated_at?:string };
export type TeacherWithdrawalRequest = { id:string; teacher_id:string; teacher_email:string; amount:number; status:"submitted"|"approved"|"rejected"|"cancelled"|"paid"; payout_method:"bank_transfer"|"e_wallet"; provider:string; account_number:string; account_holder_name:string; phone:string; admin_note:string; transfer_reference:string; proof_url:string; submitted_at:string; reviewed_at?:string; paid_at?:string; updated_at:string };
export type TeacherCommissionSummary = { upload_fees:number; sales_bonus:number; held:number; available:number; paid:number };
export type FinanceDashboardData = { overview:FinanceOverview; settings:FinanceSettings; users:UserFinanceProfile[]; commissions:TeacherCommission[]; payouts:TeacherPayout[]; payout_accounts:TeacherPayoutAccount[]; payout_requests:TeacherWithdrawalRequest[]; commission_summary:TeacherCommissionSummary };
export type AdminExam = { id: string; package_id: string; title: string; package_title: string; mapel_id: string; tahun_ajaran_id: string; duration_minutes: number; total_questions: number; passing_score: number; status: Status; publish_pembahasan: boolean; created_at: string };
export type AdminTransaction = { id: string; invoice_number: string; user_email: string; package_title: string; amount: number; status: string; created_at: string };
export type AdminPackage = { id: string; title: string; kode: string; description: string; price: number; validity_days: number; status: Status; jenjang: string; exam_type: "sell" | "cbt"; cbt_token?: string; start_date?: string; end_date?: string; kategori_id?: string; kategori_name?: string; kelas_id?: string; kelas_name?: string; publisher_email?: string; sales_count:number; view_count:number; created_at: string };
export type AdminQuestion = { id: string; exam_id: string; exam_title: string; subject_name: string; content_text: string; options: QuestionOption[]; correct_answer: string; score_weight: number; explanation_text: string; status: Status } & TKAQuestionFields;
export type Testimonial = { id: string; user_id: string; user_email: string; user_name: string; quote: string; status: "pending" | "approved" | "rejected"; created_at: string; updated_at: string };
export type MasterItem = { id: string; nama: string };
export type MasterCategory = "mapel" | "jenjang" | "tahun-ajaran" | "kategori" | "kelas";
export type AuditLog = { id: string; actor_id: string; actor_email: string; action: string; entity_type: string; entity_id: string; detail: Record<string, unknown>; created_at: string };
export type AdminPage<T> = { items: T[]; page: number; count: number; total: number };
export type TeacherVerification = { id: string; email: string; name: string; teacher_ktp: string; school_level: User["school_level"]; status: "pending" | "rejected"; rejection_reason: string; appeal_image: string; simpkb_status: "pending" | "checking" | "found" | "not_found" | "error"; simpkb_image: string; created_at: string; updated_at: string };
export type UserRoleSummary = { total: number; students: number; teachers: number; admins: number };
export type AdminUsersPage = AdminPage<User> & { summary: UserRoleSummary };
export type AdminPackagesPage = AdminPage<AdminPackage> & { counts: { active: number; inactive: number; active_teacher: number; inactive_teacher: number } };
export type CBTLookupExam = { exam_id: string; title: string; duration_minutes: number; total_questions: number; passing_score: number; submitted: boolean; publish_pembahasan: boolean; user_exam_id?: string };
export type CBTLookupPackage = { id: string; title: string; kode: string; jenjang: string; price: number; exams: CBTLookupExam[] };
export type CBTPublishSetting = { exam_id: string; package_id: string; package_title: string; package_kode: string; jenjang: string; exam_title: string; duration_minutes: number; total_questions: number; passing_score: number; publish_pembahasan: boolean; participated: number; publisher_email: string };
export type CBTParticipant = { user_exam_id: string; user_id: string; name: string; email: string; school_level: string; status: "ongoing" | "submitted"; started_at?: string; finished_at?: string; total_questions: number; current_question?: number; total_score: number; passing_score: number; passed: boolean };
export type AdminDashboardData = { overview: AdminOverview; levels: string[]; jenjangs: MasterItem[]; mapels: MasterItem[]; academic_years: MasterItem[]; kategoris: MasterItem[]; kelas: MasterItem[]; users: User[]; packages: AdminPackage[]; exams: AdminExam[]; transactions: AdminTransaction[]; questions: AdminQuestion[] };
export type TeacherOverview = { total_packages: number; total_exams: number; total_questions: number; total_sales: number; total_revenue: number };
export type TeacherDashboardData = { overview: TeacherOverview; levels: string[]; mapels: MasterItem[]; academic_years: MasterItem[]; kategoris: MasterItem[]; kelas: MasterItem[]; packages: AdminPackage[]; exams: AdminExam[]; questions: AdminQuestion[]; transactions: AdminTransaction[]; finance_settings:FinanceSettings; commission_summary:TeacherCommissionSummary; commissions:TeacherCommission[]; payouts:TeacherPayout[]; payout_account:TeacherPayoutAccount; payout_requests:TeacherWithdrawalRequest[] };
export type AffiliateReferral = { user_id:string; email:string; name:string; referred_at:string; first_purchase_at?:string; commission_id?:string; commission_amount?:number };
export type AffiliateDashboardData = { referral_code:string; referral_link:string; payout_account:TeacherPayoutAccount; referrals:AffiliateReferral[]; commissions:TeacherCommission[]; commission_summary:TeacherCommissionSummary; payouts:TeacherPayout[]; payout_requests:TeacherWithdrawalRequest[]; finance_settings:FinanceSettings };
export type ExamEntry = { title: string; mapel_id: string; tahun_ajaran_id: string; duration_minutes: number; total_questions: number; passing_score: number; status: Status };
export type QuestionEntry = { subject_name: string; content_text: string; options: QuestionOption[]; correct_answer: string; score_weight: number; explanation_text: string; status: Status } & TKAQuestionFields;
export type PackageEntry = { title: string; kode: string; description: string; price: number; validity_days: number; status: Status; jenjang: string; exam_type: "sell" | "cbt"; cbt_token?: string; start_date?: string; end_date?: string; kategori_id: string; kelas_id: string };
export type PackageBundleCreate = { package: PackageEntry; exam?: ExamEntry; questions?: QuestionEntry[] };

export const tokenStore = {
  set(token: string, user?: User) {
    localStorage.setItem(TOKEN_KEY, token);
    localStorage.removeItem(LEGACY_TOKEN_KEY);
    if (user) localStorage.setItem(USER_KEY, JSON.stringify(user));
  },
  setRefresh(refresh: string) { localStorage.setItem(REFRESH_KEY, refresh); },
  getRefresh() { return typeof window === "undefined" ? null : localStorage.getItem(REFRESH_KEY); },
  getUser(): User | null {
    if (typeof window === "undefined") return null;
    try { return JSON.parse(localStorage.getItem(USER_KEY) ?? "null") as User | null; } catch { return null; }
  },
  setUser(user: User) { localStorage.setItem(USER_KEY, JSON.stringify(user)); },
  hasToken() {
    return typeof window !== "undefined" && Boolean(localStorage.getItem(TOKEN_KEY) ?? localStorage.getItem(LEGACY_TOKEN_KEY));
  },
  clear() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(LEGACY_TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
    localStorage.removeItem(USER_KEY);
  },
};

export const api = {
  async register(email: string, password: string, schoolLevel: User["school_level"], role: "student" | "teacher" = "student", referralCode?: string, teacherKtp?: string) {
    return (await http.post<{ user: User }>("/auth/register", { email, password, school_level: schoolLevel, role, ...(role === "teacher" && teacherKtp ? { teacher_ktp: teacherKtp } : {}), ...(referralCode ? { referral_code: referralCode } : {}) })).data;
  },
  async login(email: string, password: string) {
    const response = (await http.post<LoginResponse>("/auth/login", { email, password })).data;
    if (response.access_token) { tokenStore.set(response.access_token, response.user); if (response.refresh_token) tokenStore.setRefresh(response.refresh_token); }
    return response;
  },
  async verify2FA(mfaToken: string, code: string) {
    const response = (await http.post<LoginResponse>("/auth/verify-2fa", { mfa_token: mfaToken, code })).data;
    if (response.access_token) { tokenStore.set(response.access_token, response.user); if (response.refresh_token) tokenStore.setRefresh(response.refresh_token); }
    return response;
  },
  async twoFactorSetup() { return (await http.get<{ secret: string; otpauth_url: string; qr_data_url: string; enabled: boolean }>("/auth/2fa/setup")).data; },
  async twoFactorEnable(code: string) { return (await http.post<{ totp_enabled: boolean }>("/auth/2fa/enable", { code })).data; },
  async twoFactorDisable(code: string) { return (await http.post<{ totp_enabled: boolean }>("/auth/2fa/disable", { code })).data; },
  async forgotPassword(email:string) { return (await http.post<{message:string;reset_url?:string}>("/auth/forgot-password", {email})).data; },
  async resetPassword(token:string,newPassword:string) { return (await http.post<{message:string}>("/auth/reset-password", {token,new_password:newPassword})).data; },
  async loginWithGoogle(idToken: string, schoolLevel?: User["school_level"]) {
    const response = (await http.post<LoginResponse>("/auth/google", { id_token: idToken, school_level: schoolLevel })).data;
    if (response.access_token) { tokenStore.set(response.access_token, response.user); if (response.refresh_token) tokenStore.setRefresh(response.refresh_token); }
    return response;
  },
  async logout() { try { await http.post("/auth/logout"); } finally { tokenStore.clear(); } },
  async currentUser() {
    const response = (await http.get<{ user: User }>("/auth/me")).data.user;
    tokenStore.setUser(response);
    return response;
  },
  async updateProfile(input: { name?: string; birth_date?: string; phone?: string; school_level?: User["school_level"]; current_password?: string; new_password?: string }) {
    const response = (await http.patch<{ user: User }>("/auth/me", input)).data.user;
    tokenStore.setUser(response);
    return response;
  },
  async packages() { return (await http.get<{ data: Package[] }>("/packages")).data; },
  async siteSettings() { return (await http.get<{settings:SiteSettings}>("/settings")).data.settings; },
  async trackPackageView(packageID:string) {
    let visitorKey = localStorage.getItem("tka_catalog_visitor");
    if (!visitorKey) { visitorKey = crypto.randomUUID(); localStorage.setItem("tka_catalog_visitor", visitorKey); }
    return (await http.post<{view_count:number}>(`/packages/${packageID}/view`, { visitor_key:visitorKey })).data.view_count;
  },
  async packagesMine() { return (await http.get<{ data: OwnedPackage[] }>("/packages/mine")).data.data; },
  async pricingPolicy() { return (await http.get<{ policy: UserPricingPolicy }>("/finance/me")).data.policy; },
  async packageExams(packageID: string) { return (await http.get<{ data: PackageExam[] }>(`/packages/${packageID}/exams`)).data.data; },
  async myTransactions() { return (await http.get<{ data: MyTransaction[] }>("/transactions/mine")).data.data; },
  async pendingTransactions() { return (await http.get<{ data: PendingTransaction[] }>("/transactions/pending")).data.data; },
  async downloadInvoice(transactionID: string) { return (await http.get<Blob>(`/transactions/${transactionID}/invoice.pdf`, { responseType: "blob" })).data; },
  async claimFreePackage(packageID: string) { return (await http.post<{ package: OwnedPackage }>(`/packages/${packageID}/claim`)).data.package; },
  async checkout(packageID: string, paymentMethod: "qris" | "virtual_account" | "e_wallet", idempotencyKey: string, referralCode?: string) {
    return (await http.post<CheckoutResponse>("/transactions/checkout", { package_id: packageID, payment_method: paymentMethod, ...(referralCode ? { referral_code: referralCode } : {}) }, { headers: { "Idempotency-Key": idempotencyKey } })).data;
  },
  async startExam(examID: string, cbtToken = "") { return (await http.get<ExamStartResponse>(`/exams/${examID}/start`, { params: cbtToken ? { cbt_token: cbtToken } : {} })).data; },
  async cbtLookup(token: string) { return (await http.get<{ data: CBTLookupPackage[] }>("/cbt/lookup", { params: { token } })).data.data; },
  async adminCBTSettings() { return (await http.get<{ items: CBTPublishSetting[] }>("/admin/cbt-settings")).data.items; },
  async adminSetCBTPublish(examID: string, publishPembahasan: boolean) { return (await http.patch<{ item: CBTPublishSetting }>(`/admin/cbt-settings/${examID}`, { publish_pembahasan: publishPembahasan })).data.item; },
  async adminCBTParticipants(examID: string) { return (await http.get<{ items: CBTParticipant[] }>(`/admin/cbt-settings/${examID}/participants`)).data.items; },
  async teacherCBTSettings() { return (await http.get<{ items: CBTPublishSetting[] }>("/teacher/cbt-settings")).data.items; },
  async teacherSetCBTPublish(examID: string, publishPembahasan: boolean) { return (await http.patch<{ item: CBTPublishSetting }>(`/teacher/cbt-settings/${examID}`, { publish_pembahasan: publishPembahasan })).data.item; },
  async teacherCBTParticipants(examID: string) { return (await http.get<{ items: CBTParticipant[] }>(`/teacher/cbt-settings/${examID}/participants`)).data.items; },
  async syncAnswer(examID: string, questionID: string, selectedOption: string) {
    return (await http.post<SyncAnswerResponse>("/cbt/answers/sync", { exam_id: examID, question_id: questionID, selected_option: selectedOption })).data;
  },
  async submitExam(userExamID: string) { return (await http.post<SubmitExamResponse>(`/exams/${userExamID}/submit`, {})).data; },
  async examResult(userExamID: string) { return (await http.get<ExamResult>(`/exams/${userExamID}/result`)).data; },
  async globalRanking(jenjang?: string, page: number = 1, perPage: number = 50, mode?: string) {
    const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
    if (jenjang) params.set("jenjang", jenjang);
    if (mode) params.set("mode", mode);
    return (await http.get<AdminPage<GlobalRanking>>(`/rankings/global?${params.toString()}`)).data;
  },
  async adminDashboard() { return (await http.get<AdminDashboardData>("/admin/dashboard")).data; },
  async adminUsers(page: number = 1, perPage: number = 25, filters: { role?: string; level?: string; q?: string } = {}) {
    const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
    if (filters.role) params.set("role", filters.role);
    if (filters.level) params.set("level", filters.level);
    if (filters.q) params.set("q", filters.q);
    return (await http.get<AdminUsersPage>(`/admin/users?${params.toString()}`)).data;
  },
  async adminPackages(page: number = 1, perPage: number = 25, filters: { status?: string; jenjang?: string; q?: string; examType?: string } = {}) {
    const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
    if (filters.status) params.set("status", filters.status);
    if (filters.jenjang) params.set("jenjang", filters.jenjang);
    if (filters.q) params.set("q", filters.q);
    if (filters.examType) params.set("exam_type", filters.examType);
    return (await http.get<AdminPackagesPage>(`/admin/packages?${params.toString()}`)).data;
  },
  async adminExams(packageID: string | undefined, page: number = 1, perPage: number = 25) {
    const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
    if (packageID) params.set("package_id", packageID);
    return (await http.get<AdminPage<AdminExam>>(`/admin/exams?${params.toString()}`)).data;
  },
  async adminQuestions(packageID: string | undefined, page: number = 1, perPage: number = 25) {
    const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
    if (packageID) params.set("package_id", packageID);
    return (await http.get<AdminPage<AdminQuestion>>(`/admin/questions?${params.toString()}`)).data;
  },
  async adminTransactions(page: number = 1, perPage: number = 25, status?: string) {
    const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
    if (status) params.set("status", status);
    const response = (await http.get<{ items: AdminTransaction[]; page: number; count: number; total: number }>(`/admin/transactions?${params.toString()}`)).data;
    return { ...response, transactions: response.items };
  },
  async adminRefundTransaction(id: string, reason: string) {
    return (await http.post<{ transaction: AdminTransaction }>(`/admin/transactions/${id}/refund`, { reason })).data.transaction;
  },
  async adminAudit() { return (await http.get<{items: AuditLog[]}>("/admin/audit")).data.items; },
  async updateSiteSettings(input:SiteSettings) { return (await http.put<{settings:SiteSettings}>("/admin/settings",input)).data.settings; },
  async adminFinance() { return (await http.get<FinanceDashboardData>("/admin/finance")).data; },
  async exportFinanceCSV() { return (await http.get<Blob>("/admin/finance/export", { responseType: "blob" })).data; },
  async updateFinanceSettings(input:FinanceSettings) { return (await http.put<{settings:FinanceSettings}>("/admin/finance/settings",input)).data.settings; },
  async payTeacherCommissions(id:string,reference:string) { return (await http.post<{payout:TeacherPayout}>(`/admin/finance/teachers/${id}/payout`,{reference})).data.payout; },
  async reviewPayoutRequest(id:string,input:{status:"approved"|"rejected"|"paid";note:string;reference:string;proof_url:string}) { return (await http.patch<{payout_request:TeacherWithdrawalRequest}>(`/admin/finance/payout-requests/${id}`,input)).data.payout_request; },
  async updateUserFinance(id:string,input:{commission_percent:number|null;discount_percent:number|null;account_status:"active"|"hold";notes:string}) { return (await http.put<{profile:UserFinanceProfile}>(`/admin/finance/users/${id}`,input)).data.profile; },
  async updateAdminUser(userID: string, role: User["role"], schoolLevel: User["school_level"]) {
    return (await http.patch<{ user: User }>(`/admin/users/${userID}`, { role, school_level: schoolLevel })).data.user;
  },
  async createAdminUser(input: { email: string; password: string; role: User["role"]; school_level: User["school_level"] }) { return (await http.post<{user:User}>("/admin/users",input)).data.user; },
  async deleteAdminUser(id:string) { await http.delete(`/admin/users/${id}`); },
  async createAdminPackage(input: PackageEntry) { return (await http.post<{package:AdminPackage}>("/admin/packages",input)).data.package; },
  async createAdminPackageBundle(input:PackageBundleCreate) { return (await http.post<{package:AdminPackage}>("/admin/packages/bundle",input)).data.package; },
  async updateAdminPackage(id:string,input:PackageEntry) { return (await http.put<{package:AdminPackage}>(`/admin/packages/${id}`,input)).data.package; },
  async deleteAdminPackage(id:string) { await http.delete(`/admin/packages/${id}`); },
  async createAdminExam(input:{package_id:string;title:string;duration_minutes:number;total_questions:number;passing_score:number;status:Status}) { return (await http.post<{exam:AdminExam}>("/admin/exams",input)).data.exam; },
  async updateAdminExam(id:string,input:{package_id:string;title:string;mapel_id:string;tahun_ajaran_id:string;duration_minutes:number;total_questions:number;passing_score:number;status:Status}) { return (await http.put<{exam:AdminExam}>(`/admin/exams/${id}`,input)).data.exam; },
  async deleteAdminExam(id:string) { await http.delete(`/admin/exams/${id}`); },
  async createAdminQuestion(input:QuestionEntry & {exam_id:string}) { return (await http.post<{question:AdminQuestion}>("/admin/questions",input)).data.question; },
  async updateAdminQuestion(id:string,input:QuestionEntry & {exam_id:string}) { return (await http.put<{question:AdminQuestion}>(`/admin/questions/${id}`,input)).data.question; },
  async deleteAdminQuestion(id:string) { await http.delete(`/admin/questions/${id}`); },
  async bulkDeleteAdminQuestions(ids:string[], packageId?:string) { return (await http.post<{deleted:number}>("/admin/questions/bulk-delete",{ids,package_id:packageId})).data.deleted; },
  async listMaster(category: MasterCategory) { return (await http.get<{items: MasterItem[]}>(`/admin/master/${category}`)).data.items; },
  async createMaster(category: MasterCategory, nama: string) { return (await http.post<{item: MasterItem}>(`/admin/master/${category}`, { nama })).data.item; },
  async updateMaster(category: MasterCategory, id: string, nama: string) { return (await http.put<{item: MasterItem}>(`/admin/master/${category}/${id}`, { nama })).data.item; },
  async deleteMaster(category: MasterCategory, id: string) { await http.delete(`/admin/master/${category}/${id}`); },
  async teacherDashboard() { return (await http.get<TeacherDashboardData>("/teacher/dashboard")).data; },
  async teacherAppeal(appealImageDataURL: string) { return (await http.post<{ status: "pending" }>("/teacher/appeal", { appeal_image_data_url: appealImageDataURL })).data; },
  async teacherVerifications(page: number = 1, perPage: number = 25, status?: string) {
    const params = new URLSearchParams({ page: String(page), per_page: String(perPage) });
    if (status) params.set("status", status);
    return (await http.get<AdminPage<TeacherVerification>>(`/admin/teachers/verification?${params.toString()}`)).data;
  },
  async approveTeacher(id: string) { return (await http.post<{ status: "approved" }>(`/admin/teachers/${id}/verify`)).data; },
  async rejectTeacher(id: string, reason: string) { return (await http.post<{ status: "rejected" }>(`/admin/teachers/${id}/reject`, { reason })).data; },
  async checkTeacherSIMPKB(id: string) { return (await http.post<{ status: string; message: string }>(`/admin/teachers/${id}/simpkb/check`)).data; },
  async affiliateDashboard() { return (await http.get<AffiliateDashboardData>("/affiliate/dashboard")).data; },
  async updateAffiliatePayoutAccount(input:{method:"bank_transfer"|"e_wallet";provider:string;account_number:string;account_holder_name:string;phone:string}) { return (await http.put<{payout_account:TeacherPayoutAccount}>("/affiliate/payout-account",input)).data.payout_account; },
  async createAffiliatePayoutRequest(amount:number) { return (await http.post<{payout_request:TeacherWithdrawalRequest}>("/affiliate/payout-requests",{amount})).data.payout_request; },
  async cancelAffiliatePayoutRequest(id:string) { return (await http.patch<{payout_request:TeacherWithdrawalRequest}>(`/affiliate/payout-requests/${id}/cancel`)).data.payout_request; },
  async updateTeacherPayoutAccount(input:{method:"bank_transfer"|"e_wallet";provider:string;account_number:string;account_holder_name:string;phone:string}) { return (await http.put<{payout_account:TeacherPayoutAccount}>("/teacher/payout-account",input)).data.payout_account; },
  async createTeacherPayoutRequest(amount:number) { return (await http.post<{payout_request:TeacherWithdrawalRequest}>("/teacher/payout-requests",{amount})).data.payout_request; },
  async cancelTeacherPayoutRequest(id:string) { return (await http.patch<{payout_request:TeacherWithdrawalRequest}>(`/teacher/payout-requests/${id}/cancel`)).data.payout_request; },
  async createTeacherPackage(input:PackageEntry) { return (await http.post<{package:AdminPackage}>("/teacher/packages",input)).data.package; },
  async createTeacherPackageBundle(input:PackageBundleCreate) { return (await http.post<{package:AdminPackage}>("/teacher/packages/bundle",input)).data.package; },
  async updateTeacherPackage(id:string,input:PackageEntry) { return (await http.put<{package:AdminPackage}>(`/teacher/packages/${id}`,input)).data.package; },
  async deleteTeacherPackage(id:string) { await http.delete(`/teacher/packages/${id}`); },
  async createTeacherExam(input:{package_id:string;title:string;duration_minutes:number;total_questions:number;passing_score:number;status:Status}) { return (await http.post<{exam:AdminExam}>("/teacher/exams",input)).data.exam; },
  async updateTeacherExam(id:string,input:{package_id:string;title:string;mapel_id:string;tahun_ajaran_id:string;duration_minutes:number;total_questions:number;passing_score:number;status:Status}) { return (await http.put<{exam:AdminExam}>(`/teacher/exams/${id}`,input)).data.exam; },
  async deleteTeacherExam(id:string) { await http.delete(`/teacher/exams/${id}`); },
  async createTeacherQuestion(input:QuestionEntry & {exam_id:string}) { return (await http.post<{question:AdminQuestion}>("/teacher/questions",input)).data.question; },
  async updateTeacherQuestion(id:string,input:QuestionEntry & {exam_id:string}) { return (await http.put<{question:AdminQuestion}>(`/teacher/questions/${id}`,input)).data.question; },
  async deleteTeacherQuestion(id:string) { await http.delete(`/teacher/questions/${id}`); },
  async bulkDeleteTeacherQuestions(ids:string[], packageId?:string) { return (await http.post<{deleted:number}>("/teacher/questions/bulk-delete",{ids,package_id:packageId})).data.deleted; },
  async submitTestimonial(quote: string) { return (await http.post<{testimonial: Testimonial}>("/testimonials", { quote })).data.testimonial; },
  async notifications() { return (await http.get<{ notifications: AppNotification[] }>("/notifications", { params: { limit: 50 } })).data.notifications; },
  async notificationUnreadCount() { return (await http.get<{ unread_count: number }>("/notifications/unread-count")).data.unread_count; },
  async markNotificationRead(id: string) { await http.patch(`/notifications/${id}/read`); },
  async markAllNotificationsRead() { await http.patch("/notifications/mark-all-read"); },
  async getMyTestimonial() { return (await http.get<{testimonial: Testimonial | null}>("/testimonials/mine")).data.testimonial; },
  async getPublicTestimonials() { return (await http.get<{testimonials: Testimonial[]}>("/testimonials")).data.testimonials; },
  async adminListTestimonials(status?: string) { return (await http.get<{testimonials: Testimonial[]}>(`/admin/testimonials${status ? `?status=${encodeURIComponent(status)}` : ""}`)).data.testimonials; },
  async adminUpdateTestimonialStatus(id: string, status: "pending" | "approved" | "rejected") { return (await http.patch<{testimonial: Testimonial}>(`/admin/testimonials/${id}`, { status })).data.testimonial; },
  async adminDeleteTestimonial(id: string) { await http.delete(`/admin/testimonials/${id}`); },
};
