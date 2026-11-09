package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tugas2/app/model"
	"tugas2/config"
	"tugas2/database"
	"tugas2/helper"
)

func fire(n int, job func(i int) string) map[string]int {
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make(chan string, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int, out chan<- string) {
			defer wg.Done()
			<-start
			out <- job(i)
		}(i, results)
	}
	close(start)
	wg.Wait()
	close(results)

	tally := map[string]int{}
	for r := range results {
		tally[r]++
	}
	return tally
}

type student struct {
	user  model.User
	token string
}

type env struct {
	ctx        context.Context
	pool       *pgxpool.Pool
	base       string
	client     *http.Client
	users      []student
	courseID   map[string]int
	adminToken string
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}

func main() {
	config.LoadEnv()

	mode := flag.String("mode", "crowd", "serial | crowd | credits | swap | assign")
	count := flag.Int("n", 7, "jumlah mahasiswa pada mode serial")
	base := flag.String("base", "http://localhost:"+config.GetEnv("APP_PORT", "3000"),
		"alamat dasar server")
	flag.Parse()

	secret := config.GetEnv("JWT_SECRET", "")
	if len(secret) < 32 {
		fatal("JWT_SECRET kosong atau kurang dari 32 karakter (jalankan dari root proyek agar .env terbaca)")
	}
	jwtManager := helper.NewJWTManager(
		secret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	ctx := context.Background()
	pool, err := database.NewPool(ctx)
	if err != nil {
		fatal("database: %v", err)
	}
	defer pool.Close()

	e := &env{
		ctx:  ctx,
		pool: pool,
		base: strings.TrimRight(*base, "/"),
		client: &http.Client{
			Timeout: 20 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        256,
				MaxIdleConnsPerHost: 256,
			},
		},
		courseID: map[string]int{},
	}

	if err := e.loadUsers(jwtManager); err != nil {
		fatal("%v", err)
	}
	if err := e.loadCourses(); err != nil {
		fatal("%v", err)
	}

	var consistent bool
	switch *mode {
	case "serial":
		consistent = runSerial(e, *count)
	case "crowd":
		consistent = runCrowd(e)
	case "credits":
		consistent = runCredits(e)
	case "swap":
		consistent = runSwap(e)
	case "assign":
		if err := e.loadAdmin(jwtManager); err != nil {
			fatal("%v", err)
		}
		consistent = runAssign(e)
	default:
		fatal("mode tidak dikenal: %q (pilih serial, crowd, credits, atau swap)", *mode)
	}
	if !consistent {
		os.Exit(1)
	}
}

func (e *env) loadUsers(jwtManager *helper.JWTManager) error {
	rows, err := e.pool.Query(e.ctx,
		"SELECT id, username, email, role FROM users WHERE username LIKE 'mhs%' ORDER BY username")
	if err != nil {
		return fmt.Errorf("mengambil akun uji: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Role); err != nil {
			return fmt.Errorf("membaca akun uji: %w", err)
		}
		token, err := jwtManager.GenerateAccess(u)
		if err != nil {
			return fmt.Errorf("membuat token %s: %w", u.Username, err)
		}
		e.users = append(e.users, student{user: u, token: token})
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(e.users) < 60 {
		return fmt.Errorf("hanya ada %d akun mhs%%, butuh 60 (jalankan migrations/008_seed_loadtest.sql)", len(e.users))
	}
	return nil
}

func (e *env) loadCourses() error {
	rows, err := e.pool.Query(e.ctx, "SELECT id, code FROM courses")
	if err != nil {
		return fmt.Errorf("mengambil daftar kelas: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return err
		}
		e.courseID[code] = id
	}
	return rows.Err()
}

func (e *env) call(token, method, path string, body any) string {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return "ERROR marshal"
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, e.base+path, reader)
	if err != nil {
		return "ERROR request"
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		msg := err.Error()
		if len(msg) > 50 {
			msg = msg[len(msg)-50:]
		}
		return "ERROR " + msg
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("%d OK", resp.StatusCode)
	}
	var errBody struct {
		Code string `json:"code"`
	}
	_ = json.Unmarshal(data, &errBody)
	if errBody.Code == "" {
		errBody.Code = "-"
	}
	return fmt.Sprintf("%d %s", resp.StatusCode, errBody.Code)
}

func printTally(tally map[string]int, elapsed time.Duration) {
	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Println("hasil request:")
	for _, k := range keys {
		fmt.Printf("  %-32s %d\n", k, tally[k])
	}
	fmt.Printf("waktu total : %s\n", elapsed.Round(time.Millisecond))
}

func (e *env) courseLine(code string) bool {
	var quota, enrolled, rows int
	err := e.pool.QueryRow(e.ctx,
		`SELECT c.quota, c.enrolled_count,
		        (SELECT COUNT(*) FROM enrollments WHERE course_id = c.id)
		 FROM courses c WHERE c.code = $1`, code).Scan(&quota, &enrolled, &rows)
	if err != nil {
		fmt.Printf("%s: gagal membaca (%v)\n", code, err)
		return false
	}
	ok := enrolled == rows && enrolled <= quota
	status := "KONSISTEN"
	if !ok {
		status = "TIDAK KONSISTEN"
	}
	fmt.Printf("%s kuota=%d enrolled_count=%d baris enrollments=%d -> %s\n",
		code, quota, enrolled, rows, status)
	return ok
}

func (e *env) recount() error {
	_, err := e.pool.Exec(e.ctx,
		`UPDATE courses c
		 SET enrolled_count = (SELECT COUNT(*) FROM enrollments e WHERE e.course_id = c.id)`)
	return err
}

func (e *env) enrollPath(code string) string {
	return fmt.Sprintf("/api/v1/courses/%d/enroll", e.courseID[code])
}

func runSerial(e *env, n int) bool {
	if n < 1 || n > len(e.users) {
		fatal("-n harus antara 1 dan %d", len(e.users))
	}
	if _, ok := e.courseID["IF399"]; !ok {
		fatal("kelas IF399 tidak ada (jalankan migrations/007_courses.sql)")
	}
	fmt.Printf("%d mahasiswa mendaftar ke IF399 satu per satu...\n", n)
	for i := 0; i < n; i++ {
		u := e.users[i]
		res := e.call(u.token, http.MethodPost, e.enrollPath("IF399"), nil)
		fmt.Printf("%s -> %s\n", u.user.Username, res)
	}
	return e.courseLine("IF399")
}

func runCrowd(e *env) bool {
	if _, ok := e.courseID["IF399"]; !ok {
		fatal("kelas IF399 tidak ada (jalankan migrations/007_courses.sql)")
	}
	fmt.Println("60 mahasiswa berebut kelas IF399 secara serentak...")

	start := time.Now()
	tally := fire(60, func(i int) string {
		return e.call(e.users[i].token, http.MethodPost, e.enrollPath("IF399"), nil)
	})
	elapsed := time.Since(start)

	printTally(tally, elapsed)
	return e.courseLine("IF399")
}

func runCredits(e *env) bool {
	codes := []string{
		"IF101", "IF102", "IF301", "IF302", "IF303", "IF304",
		"IF305", "IF306", "IF307", "IF308", "IF309", "IF310",
	}
	const students = 10
	fmt.Printf("%d mahasiswa, masing-masing mendaftar ke %d kelas 3 SKS secara serentak (batas 24 SKS)...\n",
		students, len(codes))

	start := time.Now()
	tally := fire(students*len(codes), func(i int) string {
		u := e.users[i/len(codes)]
		return e.call(u.token, http.MethodPost, e.enrollPath(codes[i%len(codes)]), nil)
	})
	elapsed := time.Since(start)
	printTally(tally, elapsed)

	rows, err := e.pool.Query(e.ctx,
		`SELECT u.username, COALESCE(SUM(c.credits), 0)
		 FROM users u
		 LEFT JOIN enrollments en ON en.user_id = u.id
		 LEFT JOIN courses c ON c.id = en.course_id
		 WHERE u.username LIKE 'mhs%'
		 GROUP BY u.username
		 HAVING COALESCE(SUM(c.credits), 0) > 24
		 ORDER BY u.username`)
	if err != nil {
		fmt.Printf("gagal memeriksa SKS: %v\n", err)
		return false
	}
	defer rows.Close()
	over := 0
	for rows.Next() {
		var name string
		var total int
		if err := rows.Scan(&name, &total); err != nil {
			fmt.Printf("gagal membaca SKS: %v\n", err)
			return false
		}
		fmt.Printf("  %s mengambil %d SKS (> 24)\n", name, total)
		over++
	}

	var mismatched int
	if err := e.pool.QueryRow(e.ctx,
		`SELECT COUNT(*) FROM courses c
		 WHERE c.enrolled_count <> (SELECT COUNT(*) FROM enrollments WHERE course_id = c.id)
		    OR c.enrolled_count > c.quota`).Scan(&mismatched); err != nil {
		fmt.Printf("gagal memeriksa enrolled_count: %v\n", err)
		return false
	}

	if over == 0 && mismatched == 0 {
		fmt.Printf("seluruh %d mahasiswa <= 24 SKS -> KONSISTEN\n", students)
		return true
	}
	fmt.Printf("%d mahasiswa melewati 24 SKS, %d kelas enrolled_count tidak cocok -> TIDAK KONSISTEN\n",
		over, mismatched)
	return false
}

func (e *env) deadlocks() (int64, bool) {
	var n int64
	err := e.pool.QueryRow(e.ctx,
		"SELECT deadlocks FROM pg_stat_database WHERE datname = current_database()").Scan(&n)
	return n, err == nil
}

func runSwap(e *env) bool {
	const pairs = 20
	a, b := e.courseID["IF101"], e.courseID["IF102"]
	if a == 0 || b == 0 {
		fatal("kelas IF101/IF102 tidak ada (jalankan migrations/007_courses.sql)")
	}

	if _, err := e.pool.Exec(e.ctx,
		"DELETE FROM enrollments WHERE user_id IN (SELECT id FROM users WHERE username LIKE 'mhs%')"); err != nil {
		fatal("persiapan swap: %v", err)
	}
	for i := 0; i < pairs*2; i++ {
		course := a
		if i >= pairs {
			course = b
		}
		if _, err := e.pool.Exec(e.ctx,
			"INSERT INTO enrollments (course_id, user_id) VALUES ($1, $2)",
			course, e.users[i].user.ID); err != nil {
			fatal("persiapan swap: %v", err)
		}
	}
	if err := e.recount(); err != nil {
		fatal("persiapan swap: %v", err)
	}

	before, canRead := e.deadlocks()
	fmt.Printf("%d pasang mahasiswa pindah kelas berlawanan arah secara serentak...\n", pairs)

	start := time.Now()
	tally := fire(pairs*2, func(i int) string {
		from, to := a, b
		if i >= pairs {
			from, to = b, a
		}
		return e.call(e.users[i].token, http.MethodPost, "/api/v1/enrollments/swap",
			map[string]int{"from_course_id": from, "to_course_id": to})
	})
	elapsed := time.Since(start)
	printTally(tally, elapsed)

	time.Sleep(2 * time.Second)
	if after, ok := e.deadlocks(); ok && canRead {
		fmt.Printf("deadlock terdeteksi PostgreSQL selama uji: %d\n", after-before)
	} else {
		fmt.Println("deadlock terdeteksi PostgreSQL selama uji: (tidak dapat dibaca)")
	}

	ok1 := e.courseLine("IF101")
	ok2 := e.courseLine("IF102")
	return ok1 && ok2
}

func (e *env) loadAdmin(jwtManager *helper.JWTManager) error {
	var u model.User
	err := e.pool.QueryRow(e.ctx,
		"SELECT id, username, email, role FROM users WHERE role = 'admin' ORDER BY id LIMIT 1").
		Scan(&u.ID, &u.Username, &u.Email, &u.Role)
	if err != nil {
		return fmt.Errorf("mengambil akun admin: %w", err)
	}
	token, err := jwtManager.GenerateAccess(u)
	if err != nil {
		return fmt.Errorf("membuat token admin: %w", err)
	}
	e.adminToken = token
	return nil
}

func runAssign(e *env) bool {
	const n = 30
	const quota = 3
	const supervisorName = "Dosen Uji Rebutan"

	for i := 1; i <= n; i++ {
		if _, err := e.pool.Exec(e.ctx,
			"INSERT INTO students (nim, name) VALUES ($1, $2) ON CONFLICT DO NOTHING",
			fmt.Sprintf("LT%03d", i), fmt.Sprintf("Mahasiswa Beban %03d", i)); err != nil {
			fatal("persiapan assign: %v", err)
		}
	}
	if _, err := e.pool.Exec(e.ctx,
		"DELETE FROM supervisions WHERE student_id IN (SELECT id FROM students WHERE nim LIKE 'LT%')"); err != nil {
		fatal("persiapan assign: %v", err)
	}
	if _, err := e.pool.Exec(e.ctx,
		`UPDATE supervisors s
		 SET supervised_count = (SELECT COUNT(*) FROM supervisions x WHERE x.supervisor_id = s.id)`); err != nil {
		fatal("persiapan assign: %v", err)
	}
	if _, err := e.pool.Exec(e.ctx,
		"UPDATE supervisors SET quota = $1 WHERE name = $2", quota, supervisorName); err != nil {
		fatal("persiapan assign: %v", err)
	}

	var supervisorID int
	if err := e.pool.QueryRow(e.ctx,
		"SELECT id FROM supervisors WHERE name = $1", supervisorName).Scan(&supervisorID); err != nil {
		fatal("pembimbing %q tidak ada (jalankan migrations/009_supervisors.sql): %v", supervisorName, err)
	}

	rows, err := e.pool.Query(e.ctx,
		"SELECT id FROM students WHERE nim LIKE 'LT%' ORDER BY nim")
	if err != nil {
		fatal("persiapan assign: %v", err)
	}
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			fatal("persiapan assign: %v", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) < n {
		fatal("hanya ada %d mahasiswa uji, butuh %d", len(ids), n)
	}

	fmt.Printf("%d penetapan serentak ke pembimbing %q berkuota %d...\n", n, supervisorName, quota)
	start := time.Now()
	tally := fire(n, func(i int) string {
		return e.call(e.adminToken, http.MethodPost,
			fmt.Sprintf("/api/v1/supervisors/%d/assign", supervisorID),
			map[string]int{"student_id": ids[i]})
	})
	elapsed := time.Since(start)
	printTally(tally, elapsed)

	var supQuota, supCount, supRows int
	if err := e.pool.QueryRow(e.ctx,
		`SELECT s.quota, s.supervised_count,
		        (SELECT COUNT(*) FROM supervisions WHERE supervisor_id = s.id)
		 FROM supervisors s WHERE s.id = $1`, supervisorID).Scan(&supQuota, &supCount, &supRows); err != nil {
		fmt.Printf("gagal membaca pembimbing: %v\n", err)
		return false
	}
	consistent := supCount == supRows && supCount <= supQuota
	status := "KONSISTEN"
	if !consistent {
		status = "TIDAK KONSISTEN"
	}
	fmt.Printf("%s kuota=%d supervised_count=%d baris supervisions=%d -> %s\n",
		supervisorName, supQuota, supCount, supRows, status)

	success := tally["201 OK"]
	full := tally["409 SUPERVISOR_FULL"]
	if success == quota && full == n-quota && consistent {
		fmt.Printf("tepat %d berhasil, %d ditolak SUPERVISOR_FULL -> LOLOS\n", quota, n-quota)
		return true
	}
	fmt.Printf("GAGAL: harusnya tepat %d berhasil (201) dan %d ditolak SUPERVISOR_FULL (409), dapat %d dan %d\n",
		quota, n-quota, success, full)
	return false
}
