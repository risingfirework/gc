import axios, { AxiosError } from "axios";

const TOKEN_KEY = "tka_access_token";
const LEGACY_TOKEN_KEY = "tka_token";
const USER_KEY = "tka_current_user";
export const SESSION_INVALID_EVENT = "tka:session-invalid";

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
    const isCredentialRequest = requestPath.endsWith("/auth/login") || requestPath.endsWith("/auth/register");
    if (status === 401 && !isCredentialRequest && typeof window !== "undefined") {
      localStorage.removeItem(TOKEN_KEY);
      localStorage.removeItem(LEGACY_TOKEN_KEY);
      localStorage.removeItem(USER_KEY);
      window.dispatchEvent(new CustomEvent(SESSION_INVALID_EVENT, { detail: message }));
      if (!window.location.pathname.endsWith("/login")) window.location.replace("/login?reason=session-expired");
    }
    return Promise.reject(new APIError(message, status, body?.code, networkError));
  },
);

export type User = { id: string; email: string; name: string; birth_date: string; phone: string; role: "student" | "admin" | "teacher"; school_level: "SD" | "SMP" | "SMA"; created_at: string; updated_at: string };
export type Status = "active" | "inactive";
export type LoginResponse = { access_token: string; token_type: "Bearer"; expires_in: number; user: User };
export type QuestionOption = { key: string; content: string; image_url?: string };
export type QuestionType = "single_choice" | "multiple_choice" | "category";
export type PresentationType = "single" | "group";
export type TKAQuestionFields = { question_type: QuestionType; presentation_type: PresentationType; group_code: string; stimulus_text: string; question_image_url: string; stimulus_image_url: string; category_labels: string[] };
export type ExamQuestion = { id: string; subject_name: string; content_text: string; options: QuestionOption[] } & TKAQuestionFields;
export type ExamStartResponse = { user_exam_id: string; exam_id: string; title: string; status: "ongoing"; server_time: string; started_at: string; ends_at: string; remaining_seconds: number; duration_minutes: number; questions: ExamQuestion[] };
export type SyncAnswerResponse = { saved: boolean; synced_at: string; remaining_seconds: number };
export type SubmitExamResponse = { user_exam_id: string; exam_id: string; status: "submitted"; finished_at: string; total_score: number; passing_score: number; passed: boolean };
export type Package = { id: string; title: string; kode: string; description: string; price: number; validity_days: number; status: Status; jenjang: string; publisher_email?: string; sales_count:number; view_count:number; created_at: string };
export type HeroSlide = { id:string; eyebrow:string; title:string; lead:string; stats:string[][] };
export type SiteSettings = { platform_name:string; platform_tagline:string; logo_data_url:string; support_email:string; whatsapp:string; instagram_url:string; youtube_url:string; hero_interval_ms:number; catalog_interval_ms:number; default_package_validity_days:number; default_exam_duration_minutes:number; default_exam_total_questions:number; default_passing_score:number; hero_slides:HeroSlide[]; updated_at?:string };
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
export type CheckoutResponse = { transaction: Transaction; payment_url: string; expires_at: string };
export type UserPricingPolicy = { discount_percent: number; account_active: boolean };
export type SubjectResult = { subject_name: string; correct_answers: number; wrong_answers: number; unanswered: number; total_questions: number; score: number };
export type AnswerReview = { question_id: string; subject_name: string; content_text: string; question_type: "single_choice" | "multiple_choice" | "category"; presentation_type: "single" | "group"; group_code?: string; stimulus_text?: string; question_image_url?: string; stimulus_image_url?: string; category_labels?: string[]; options: QuestionOption[]; selected_option?: string; correct_answer: string; is_correct: boolean; score_weight: number; explanation_text: string; explanation_video_url?: string };
export type ExamResult = { user_exam_id: string; exam_id: string; title: string; status: "submitted"; scoring_method: "standard" | "irt_2pl"; started_at: string; finished_at: string; total_score: number; passing_score: number; passed: boolean; correct_answers: number; wrong_answers: number; unanswered: number; subjects: SubjectResult[]; review: AnswerReview[] };
export type GlobalRanking = { rank: number; display_name: string; school_level: User["school_level"]; score: number; duration_seconds: number; finished_at: string; is_current_user: boolean };
export type AdminOverview = { total_users: number; total_students: number; total_packages: number; total_exams: number; total_transactions: number; paid_transactions: number };
export type FinanceSettings = { platform_commission_percent:number; default_discount_percent:number; tax_percent:number; minimum_payout:number; payout_cycle:"weekly"|"monthly"|"manual"; auto_payout:boolean; teacher_upload_fee:number; teacher_sales_bonus_percent:number; commission_hold_days:number; updated_at?:string };
export type UserFinanceProfile = { user_id:string; email:string; name:string; role:User["role"]; commission_percent?:number|null; discount_percent?:number|null; effective_commission_percent:number; effective_discount_percent:number; account_status:"active"|"hold"; notes:string; total_spend:number; gross_revenue:number; transaction_count:number; updated_at:string };
export type FinanceOverview = { gross_revenue:number; pending_revenue:number; platform_commission:number; estimated_payouts:number; paid_transactions:number };
export type TeacherCommission = { id:string; teacher_id:string; teacher_email:string; package_id:string; package_title:string; invoice_number?:string; kind:"upload_fee"|"sales_bonus"|"refund_reversal"; base_amount:number; rate_percent:number; amount:number; status:"pending"|"available"|"paid"|"cancelled"; available_at:string; paid_at?:string; payout_reference?:string; created_at:string };
export type TeacherPayout = { id:string; teacher_id:string; teacher_email:string; amount:number; reference:string; paid_at:string };
export type TeacherPayoutAccount = { teacher_id:string; method:"bank_transfer"|"e_wallet"|""; provider:string; account_number:string; account_holder_name:string; phone:string; updated_at?:string };
export type TeacherWithdrawalRequest = { id:string; teacher_id:string; teacher_email:string; amount:number; status:"submitted"|"approved"|"rejected"|"cancelled"|"paid"; payout_method:"bank_transfer"|"e_wallet"; provider:string; account_number:string; account_holder_name:string; phone:string; admin_note:string; transfer_reference:string; submitted_at:string; reviewed_at?:string; paid_at?:string; updated_at:string };
export type TeacherCommissionSummary = { upload_fees:number; sales_bonus:number; held:number; available:number; paid:number };
export type FinanceDashboardData = { overview:FinanceOverview; settings:FinanceSettings; users:UserFinanceProfile[]; commissions:TeacherCommission[]; payouts:TeacherPayout[]; payout_accounts:TeacherPayoutAccount[]; payout_requests:TeacherWithdrawalRequest[]; commission_summary:TeacherCommissionSummary };
export type AdminExam = { id: string; package_id: string; title: string; package_title: string; mapel_id: string; tahun_ajaran_id: string; duration_minutes: number; total_questions: number; passing_score: number; status: Status; created_at: string };
export type AdminTransaction = { id: string; invoice_number: string; user_email: string; package_title: string; amount: number; status: string; created_at: string };
export type AdminPackage = { id: string; title: string; kode: string; description: string; price: number; validity_days: number; status: Status; jenjang: string; publisher_email?: string; sales_count:number; view_count:number; created_at: string };
export type AdminQuestion = { id: string; exam_id: string; exam_title: string; subject_name: string; content_text: string; options: QuestionOption[]; correct_answer: string; score_weight: number; explanation_text: string; status: Status } & TKAQuestionFields;
export type Testimonial = { id: string; user_id: string; user_email: string; user_name: string; quote: string; status: "pending" | "approved" | "rejected"; created_at: string; updated_at: string };
export type MasterItem = { id: string; nama: string };
export type MasterCategory = "mapel" | "jenjang" | "tahun-ajaran";
export type AdminDashboardData = { overview: AdminOverview; levels: string[]; jenjangs: MasterItem[]; mapels: MasterItem[]; academic_years: MasterItem[]; users: User[]; packages: AdminPackage[]; exams: AdminExam[]; transactions: AdminTransaction[]; questions: AdminQuestion[] };
export type TeacherOverview = { total_packages: number; total_exams: number; total_questions: number; total_sales: number; total_revenue: number };
export type TeacherDashboardData = { overview: TeacherOverview; levels: string[]; mapels: MasterItem[]; academic_years: MasterItem[]; packages: AdminPackage[]; exams: AdminExam[]; questions: AdminQuestion[]; transactions: AdminTransaction[]; finance_settings:FinanceSettings; commission_summary:TeacherCommissionSummary; commissions:TeacherCommission[]; payouts:TeacherPayout[]; payout_account:TeacherPayoutAccount; payout_requests:TeacherWithdrawalRequest[] };
export type ExamEntry = { title: string; mapel_id: string; tahun_ajaran_id: string; duration_minutes: number; total_questions: number; passing_score: number; status: Status };
export type QuestionEntry = { subject_name: string; content_text: string; options: QuestionOption[]; correct_answer: string; score_weight: number; explanation_text: string; status: Status } & TKAQuestionFields;
export type PackageEntry = { title: string; kode: string; description: string; price: number; validity_days: number; status: Status; jenjang: string };
export type PackageBundleCreate = { package: PackageEntry; exam?: ExamEntry; questions?: QuestionEntry[] };

export const tokenStore = {
  set(token: string, user?: User) {
    localStorage.setItem(TOKEN_KEY, token);
    localStorage.removeItem(LEGACY_TOKEN_KEY);
    if (user) localStorage.setItem(USER_KEY, JSON.stringify(user));
  },
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
    localStorage.removeItem(USER_KEY);
  },
};

export const api = {
  async register(email: string, password: string, schoolLevel: User["school_level"], role: "student" | "teacher" = "student") {
    return (await http.post<{ user: User }>("/auth/register", { email, password, school_level: schoolLevel, role })).data;
  },
  async login(email: string, password: string) {
    const response = (await http.post<LoginResponse>("/auth/login", { email, password })).data;
    tokenStore.set(response.access_token, response.user);
    return response;
  },
  async forgotPassword(email:string) { return (await http.post<{message:string;reset_url?:string}>("/auth/forgot-password", {email})).data; },
  async resetPassword(token:string,newPassword:string) { return (await http.post<{message:string}>("/auth/reset-password", {token,new_password:newPassword})).data; },
  async loginWithGoogle(idToken: string, schoolLevel?: User["school_level"]) {
    const response = (await http.post<LoginResponse>("/auth/google", { id_token: idToken, school_level: schoolLevel })).data;
    tokenStore.set(response.access_token, response.user);
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
  async claimFreePackage(packageID: string) { return (await http.post<{ package: OwnedPackage }>(`/packages/${packageID}/claim`)).data.package; },
  async checkout(packageID: string, paymentMethod: "qris" | "virtual_account" | "e_wallet", idempotencyKey: string) {
    return (await http.post<CheckoutResponse>("/transactions/checkout", { package_id: packageID, payment_method: paymentMethod }, { headers: { "Idempotency-Key": idempotencyKey } })).data;
  },
  async startExam(examID: string) { return (await http.get<ExamStartResponse>(`/exams/${examID}/start`)).data; },
  async syncAnswer(examID: string, questionID: string, selectedOption: string) {
    return (await http.post<SyncAnswerResponse>("/cbt/answers/sync", { exam_id: examID, question_id: questionID, selected_option: selectedOption })).data;
  },
  async submitExam(userExamID: string) { return (await http.post<SubmitExamResponse>(`/exams/${userExamID}/submit`, {})).data; },
  async examResult(userExamID: string) { return (await http.get<ExamResult>(`/exams/${userExamID}/result`)).data; },
  async globalRanking(jenjang?: string) {
    return (await http.get<{ data: GlobalRanking[] }>(`/rankings/global${jenjang ? `?jenjang=${encodeURIComponent(jenjang)}` : ""}`)).data.data;
  },
  async adminDashboard() { return (await http.get<AdminDashboardData>("/admin/dashboard")).data; },
  async updateSiteSettings(input:SiteSettings) { return (await http.put<{settings:SiteSettings}>("/admin/settings",input)).data.settings; },
  async adminFinance() { return (await http.get<FinanceDashboardData>("/admin/finance")).data; },
  async updateFinanceSettings(input:FinanceSettings) { return (await http.put<{settings:FinanceSettings}>("/admin/finance/settings",input)).data.settings; },
  async payTeacherCommissions(id:string,reference:string) { return (await http.post<{payout:TeacherPayout}>(`/admin/finance/teachers/${id}/payout`,{reference})).data.payout; },
  async reviewPayoutRequest(id:string,input:{status:"approved"|"rejected"|"paid";note:string;reference:string}) { return (await http.patch<{payout_request:TeacherWithdrawalRequest}>(`/admin/finance/payout-requests/${id}`,input)).data.payout_request; },
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
  async listMaster(category: MasterCategory) { return (await http.get<{items: MasterItem[]}>(`/admin/master/${category}`)).data.items; },
  async createMaster(category: MasterCategory, nama: string) { return (await http.post<{item: MasterItem}>(`/admin/master/${category}`, { nama })).data.item; },
  async updateMaster(category: MasterCategory, id: string, nama: string) { return (await http.put<{item: MasterItem}>(`/admin/master/${category}/${id}`, { nama })).data.item; },
  async deleteMaster(category: MasterCategory, id: string) { await http.delete(`/admin/master/${category}/${id}`); },
  async teacherDashboard() { return (await http.get<TeacherDashboardData>("/teacher/dashboard")).data; },
  async updateTeacherPayoutAccount(input:{method:"bank_transfer"|"e_wallet";provider:string;account_number:string;account_holder_name:string;phone:string}) { return (await http.put<{payout_account:TeacherPayoutAccount}>("/teacher/payout-account",input)).data.payout_account; },
  async createTeacherPayoutRequest() { return (await http.post<{payout_request:TeacherWithdrawalRequest}>("/teacher/payout-requests")).data.payout_request; },
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
  async submitTestimonial(quote: string) { return (await http.post<{testimonial: Testimonial}>("/testimonials", { quote })).data.testimonial; },
  async getMyTestimonial() { return (await http.get<{testimonial: Testimonial | null}>("/testimonials/mine")).data.testimonial; },
  async getPublicTestimonials() { return (await http.get<{testimonials: Testimonial[]}>("/testimonials")).data.testimonials; },
  async adminListTestimonials(status?: string) { return (await http.get<{testimonials: Testimonial[]}>(`/admin/testimonials${status ? `?status=${encodeURIComponent(status)}` : ""}`)).data.testimonials; },
  async adminUpdateTestimonialStatus(id: string, status: "pending" | "approved" | "rejected") { return (await http.patch<{testimonial: Testimonial}>(`/admin/testimonials/${id}`, { status })).data.testimonial; },
  async adminDeleteTestimonial(id: string) { await http.delete(`/admin/testimonials/${id}`); },
};
