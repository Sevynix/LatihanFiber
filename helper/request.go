func ParseCursorQuery(c *fiber.Ctx) (model.CursorQuery, error) {
	limit := c.QueryInt("limit", 10)
	if limit < 1 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	q := model.CursorQuery{
		Limit:  limit,
		Search: strings.TrimSpace(c.Query("search")),
	}
	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}
	if raw := c.Query("cursor"); raw != "" {
		cursor, err := DecodeCursor(raw)
		if err != nil {
			return model.CursorQuery{}, err
		}
		q.After = &cursor
	}
	return q, nil
}