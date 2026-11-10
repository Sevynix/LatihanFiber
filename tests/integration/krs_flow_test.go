//go:build integration

package integration_test

import (
	"net/http"
	"strconv"
	"testing"

	"tugas2/app/model"
)

func enrolledCount(t *testing.T, courseID int) int {
	t.Helper()
	return countRows(t, `SELECT enrolled_count FROM courses WHERE id = $1`, courseID)
}

func eventsOf(t *testing.T, userID int, eventType string) int {
	t.Helper()
	return countRows(t,
		`SELECT COUNT(*) FROM outbox WHERE (payload->>'user_id')::INT = $1 AND event_type = $2`,
		userID, eventType)
}

func TestSwapEndpoint_TargetFullChangesNothing(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)
	from := seedCourse(t, 3, 40)
	to := seedCourse(t, 3, 1)

	if status, _ := a.enroll(t, student, from.ID); status != http.StatusCreated {
		t.Fatalf("mendaftar ke kelas asal: status = %d, seharusnya 201", status)
	}

	if _, err := pool.Exec(t.Context(), `UPDATE courses SET enrolled_count = quota WHERE id = $1`, to.ID); err != nil {
		t.Fatalf("mengisi kelas tujuan sampai penuh: %v", err)
	}

	body := `{"from_course_id": ` + strconv.Itoa(from.ID) + `, "to_course_id": ` + strconv.Itoa(to.ID) + `}`
	status, code := a.call(t, student, http.MethodPost, "/api/v1/enrollments/swap", body)
	if status != http.StatusConflict || code != "COURSE_FULL" {
		t.Fatalf("dapat %d %s, seharusnya 409 COURSE_FULL", status, code)
	}

	if n := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student.ID, from.ID); n != 1 {
		t.Errorf("pendaftaran di kelas asal = %d, seharusnya tetap 1", n)
	}
	if n := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student.ID, to.ID); n != 0 {
		t.Errorf("pendaftaran di kelas tujuan = %d, seharusnya 0", n)
	}
	if n := enrolledCount(t, from.ID); n != 1 {
		t.Errorf("enrolled_count kelas asal = %d, seharusnya tetap 1", n)
	}
	if n := enrolledCount(t, to.ID); n != 1 {
		t.Errorf("enrolled_count kelas tujuan = %d, seharusnya tetap 1 (penuh)", n)
	}
	if n := eventsOf(t, student.ID, model.EventEnrollmentSwapped); n != 0 {
		t.Errorf("event enrollment.swapped = %d, seharusnya 0", n)
	}
}

func TestDropEndpoint_RemovesEnrollmentAndWritesOneEvent(t *testing.T) {
	a := newAPI(t)
	student := seedStudent(t)
	course := seedCourse(t, 3, 40)

	if status, _ := a.enroll(t, student, course.ID); status != http.StatusCreated {
		t.Fatalf("mendaftar: status = %d, seharusnya 201", status)
	}
	if n := enrolledCount(t, course.ID); n != 1 {
		t.Fatalf("enrolled_count sebelum keluar = %d, seharusnya 1", n)
	}

	status, _ := a.call(t, student, http.MethodDelete, "/api/v1/courses/"+strconv.Itoa(course.ID)+"/enroll", "")
	if status != http.StatusNoContent {
		t.Fatalf("status = %d, seharusnya 204", status)
	}

	if n := countRows(t, `SELECT COUNT(*) FROM enrollments WHERE user_id = $1 AND course_id = $2`, student.ID, course.ID); n != 0 {
		t.Errorf("baris enrollment = %d, seharusnya 0", n)
	}
	if n := enrolledCount(t, course.ID); n != 0 {
		t.Errorf("enrolled_count = %d, seharusnya 0 (berkurang satu)", n)
	}
	if n := eventsOf(t, student.ID, model.EventEnrollmentDropped); n != 1 {
		t.Errorf("event enrollment.dropped = %d, seharusnya tepat 1", n)
	}
}
