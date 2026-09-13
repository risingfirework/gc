package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"tka/apps/backend/internal/domain"
)

// createPackageBundle membuat paket beserta ujian pertama dan daftar soal
// inline dalam satu transaksi. publisherID nil berarti paket platform/admin.
func createPackageBundle(ctx context.Context, db *pgxpool.Pool, publisherID *string, input domain.PackageBundle) (*domain.AdminPackage, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin package bundle: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var pkg domain.AdminPackage
	err = tx.QueryRow(ctx, `INSERT INTO packages(title,description,price,validity_days,status,publisher_id,kode,jenjang,exam_type,cbt_token,start_date,end_date,kategori_id,kelas_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::timestamptz,$12::timestamptz,$13::uuid,$14::uuid) RETURNING `+adminPackageColumns,
		input.Package.Title, input.Package.Description, input.Package.Price, input.Package.ValidityDays, input.Package.Status, publisherID, nullableKode(input.Package.Kode), input.Package.Jenjang, input.Package.ExamType, input.Package.CBTToken, nullableTimestamptz(input.Package.StartDate), nullableTimestamptz(input.Package.EndDate), input.Package.KategoriID, input.Package.KelasID,
	).Scan(&pkg.ID, &pkg.Title, &pkg.Description, &pkg.Price, &pkg.ValidityDays, &pkg.Status, &pkg.Kode, &pkg.Jenjang, &pkg.ExamType, &pkg.CBTToken, &pkg.StartDate, &pkg.EndDate, &pkg.KategoriID, &pkg.KelasID, &pkg.PublisherID, &pkg.CreatedAt)
	if err != nil {
		return nil, adminMutationError(err)
	}

	if input.Exam != nil {
		var examID string
		err = tx.QueryRow(ctx, `INSERT INTO exams(package_id,title,mapel_id,tahun_ajaran_id,duration_minutes,total_questions,passing_score,status) VALUES($1,$2,NULLIF($3,'')::uuid,NULLIF($4,'')::uuid,$5,$6,$7,$8) RETURNING id`,
			pkg.ID, input.Exam.Title, input.Exam.MapelID, input.Exam.TahunAjaranID, input.Exam.DurationMinutes, input.Exam.TotalQuestions, input.Exam.PassingScore, input.Exam.Status,
		).Scan(&examID)
		if err != nil {
			return nil, adminMutationError(err)
		}
		for _, question := range input.Questions {
			optionsJSON, marshalErr := json.Marshal(question.Options)
			if marshalErr != nil {
				return nil, fmt.Errorf("encode question options: %w", marshalErr)
			}
			categoryLabelsJSON, marshalErr := json.Marshal(question.CategoryLabels)
			if marshalErr != nil {
				return nil, fmt.Errorf("encode category labels: %w", marshalErr)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO questions(exam_id,subject_name,content_text,question_type,presentation_type,group_code,stimulus_text,question_image_url,stimulus_image_url,category_labels_json,options_json,correct_answer,score_weight,explanation_text,status) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
				examID, question.SubjectName, question.ContentText, question.QuestionType, question.PresentationType, question.GroupCode, question.StimulusText, question.QuestionImageURL, question.StimulusImageURL, categoryLabelsJSON, optionsJSON, question.CorrectAnswer, question.ScoreWeight, question.Explanation, question.Status,
			); err != nil {
				return nil, adminMutationError(err)
			}
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit package bundle: %w", err)
	}
	return &pkg, nil
}

func (r *AdminRepository) CreatePackageBundle(ctx context.Context, input domain.PackageBundle) (*domain.AdminPackage, error) {
	return createPackageBundle(ctx, r.db, nil, input)
}

func (r *TeacherRepository) CreatePackageBundle(ctx context.Context, publisherID string, input domain.PackageBundle) (*domain.AdminPackage, error) {
	return createPackageBundle(ctx, r.db, &publisherID, input)
}

var _ = (*AdminRepository).CreatePackageBundle
var _ = (*TeacherRepository).CreatePackageBundle
