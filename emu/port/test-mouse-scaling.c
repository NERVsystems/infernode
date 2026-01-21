/*
 * Regression tests for SDL3 mouse coordinate scaling
 *
 * Tests the coordinate transformation logic used in draw-sdl3.c
 * to ensure mouse clicks map correctly in all display modes:
 *   - Windowed mode (1:1 scaling)
 *   - HiDPI/Retina mode (2x scaling)
 *   - Full-screen mode with letterboxing (centered, aspect-ratio preserved)
 *
 * Build: cc -o test-mouse-scaling test-mouse-scaling.c
 * Run:   ./test-mouse-scaling
 */

#include <stdio.h>
#include <stdlib.h>

static int tests_run = 0;
static int tests_passed = 0;

/*
 * Destination rectangle for centered rendering (matches draw-sdl3.c)
 */
typedef struct {
	float x, y, w, h;
} FRect;

/*
 * Calculate destination rectangle for centered, aspect-ratio-preserving render.
 * This is the exact algorithm from draw-sdl3.c calc_dest_rect().
 */
static void
calc_dest_rect(int window_width, int window_height, int tex_w, int tex_h, FRect *dest)
{
	float scale_x, scale_y, scale;
	float dest_w, dest_h;

	if (window_width <= 0 || window_height <= 0 || tex_w <= 0 || tex_h <= 0) {
		dest->x = 0;
		dest->y = 0;
		dest->w = (float)tex_w;
		dest->h = (float)tex_h;
		return;
	}

	/* Calculate scale to fit texture in window while maintaining aspect ratio */
	scale_x = (float)window_width / (float)tex_w;
	scale_y = (float)window_height / (float)tex_h;
	scale = (scale_x < scale_y) ? scale_x : scale_y;

	/* Calculate destination size */
	dest_w = (float)tex_w * scale;
	dest_h = (float)tex_h * scale;

	/* Center in window */
	dest->x = ((float)window_width - dest_w) / 2.0f;
	dest->y = ((float)window_height - dest_h) / 2.0f;
	dest->w = dest_w;
	dest->h = dest_h;
}

/*
 * Transform window mouse coordinates to texture coordinates.
 * This is the exact algorithm from draw-sdl3.c window_to_texture_coords().
 */
static void
window_to_texture_coords(float win_x, float win_y, FRect *dest,
                         int tex_w, int tex_h,
                         int *tex_x, int *tex_y)
{
	float rel_x, rel_y;
	int x, y;

	if (dest->w <= 0 || dest->h <= 0) {
		*tex_x = (int)win_x;
		*tex_y = (int)win_y;
		return;
	}

	/* Subtract letterbox offset to get position relative to rendered texture */
	rel_x = win_x - dest->x;
	rel_y = win_y - dest->y;

	/* Scale from rendered size to texture size */
	x = (int)(rel_x * (float)tex_w / dest->w);
	y = (int)(rel_y * (float)tex_h / dest->h);

	/* Clamp to texture bounds */
	if (x < 0) x = 0;
	if (y < 0) y = 0;
	if (x >= tex_w) x = tex_w - 1;
	if (y >= tex_h) y = tex_h - 1;

	*tex_x = x;
	*tex_y = y;
}

/*
 * Test helper - check if result matches expected within tolerance
 */
static int
check(const char *name, int got_x, int got_y, int expect_x, int expect_y)
{
	int pass;

	tests_run++;

	/* Allow 1 pixel tolerance for rounding */
	pass = (abs(got_x - expect_x) <= 1) && (abs(got_y - expect_y) <= 1);

	if (pass) {
		tests_passed++;
		printf("PASS: %s\n", name);
	} else {
		printf("FAIL: %s\n", name);
		printf("      expected (%d, %d), got (%d, %d)\n",
		       expect_x, expect_y, got_x, got_y);
	}

	return pass;
}

/*
 * Test: Windowed mode with 1:1 scaling (window == texture size)
 */
static void
test_windowed_1to1(void)
{
	FRect dest;
	int mx, my;

	printf("\n=== Test: Windowed 1:1 scaling ===\n");

	/* 1024x768 window, 1024x768 texture - no letterboxing needed */
	calc_dest_rect(1024, 768, 1024, 768, &dest);
	printf("dest_rect: (%.1f, %.1f, %.1f, %.1f)\n", dest.x, dest.y, dest.w, dest.h);

	window_to_texture_coords(512.0f, 384.0f, &dest, 1024, 768, &mx, &my);
	check("center click", mx, my, 512, 384);

	window_to_texture_coords(0.0f, 0.0f, &dest, 1024, 768, &mx, &my);
	check("top-left corner", mx, my, 0, 0);

	window_to_texture_coords(1023.0f, 767.0f, &dest, 1024, 768, &mx, &my);
	check("bottom-right corner", mx, my, 1023, 767);
}

/*
 * Test: HiDPI/Retina mode - texture is 2x the window size in pixels
 * Window is 1024x768 logical, texture is 2048x1536 physical
 * dest_rect should fill the entire window (no letterboxing)
 */
static void
test_hidpi_2x(void)
{
	FRect dest;
	int mx, my;

	printf("\n=== Test: HiDPI 2x scaling ===\n");

	/* 1024x768 window, 2048x1536 texture (2x HiDPI) */
	calc_dest_rect(1024, 768, 2048, 1536, &dest);
	printf("dest_rect: (%.1f, %.1f, %.1f, %.1f)\n", dest.x, dest.y, dest.w, dest.h);

	window_to_texture_coords(512.0f, 384.0f, &dest, 2048, 1536, &mx, &my);
	check("center click", mx, my, 1024, 768);

	window_to_texture_coords(0.0f, 0.0f, &dest, 2048, 1536, &mx, &my);
	check("top-left corner", mx, my, 0, 0);

	window_to_texture_coords(1023.0f, 767.0f, &dest, 2048, 1536, &mx, &my);
	check("bottom-right corner", mx, my, 2046, 1534);
}

/*
 * Test: Full-screen mode with letterboxing
 * Texture maintains original size, centered in larger window
 */
static void
test_fullscreen_letterbox(void)
{
	FRect dest;
	int mx, my;

	printf("\n=== Test: Full-screen with letterboxing ===\n");

	/*
	 * Scenario: 2048x1536 texture (4:3 aspect) in 2560x1600 window (16:10 aspect)
	 * The texture should be scaled to fit height, with pillarboxing on sides.
	 * Scale factor: 1600/1536 = 1.0417
	 * Rendered size: 2048*1.0417 x 1536*1.0417 = 2133.3 x 1600
	 * Pillarbox: (2560-2133.3)/2 = 213.3 pixels on each side
	 */
	calc_dest_rect(2560, 1600, 2048, 1536, &dest);
	printf("dest_rect: (%.1f, %.1f, %.1f, %.1f)\n", dest.x, dest.y, dest.w, dest.h);

	/* Click in center of window = center of texture */
	window_to_texture_coords(1280.0f, 800.0f, &dest, 2048, 1536, &mx, &my);
	check("center click", mx, my, 1024, 768);

	/* Click at top-left of rendered area (not window corner) */
	window_to_texture_coords(dest.x, dest.y, &dest, 2048, 1536, &mx, &my);
	check("top-left of texture", mx, my, 0, 0);

	/* Click at bottom-right of rendered area */
	window_to_texture_coords(dest.x + dest.w - 1, dest.y + dest.h - 1, &dest, 2048, 1536, &mx, &my);
	check("bottom-right of texture", mx, my, 2047, 1535);

	/* Click in letterbox area (pillarbox) - should clamp to edge */
	window_to_texture_coords(0.0f, 800.0f, &dest, 2048, 1536, &mx, &my);
	check("click in left pillarbox (clamped)", mx, my, 0, 768);

	window_to_texture_coords(2559.0f, 800.0f, &dest, 2048, 1536, &mx, &my);
	check("click in right pillarbox (clamped)", mx, my, 2047, 768);
}

/*
 * Test: Full-screen with top/bottom letterboxing (wide texture in tall window)
 */
static void
test_fullscreen_letterbox_vertical(void)
{
	FRect dest;
	int mx, my;

	printf("\n=== Test: Full-screen with vertical letterboxing ===\n");

	/*
	 * Scenario: 1920x1080 texture (16:9) in 1600x1200 window (4:3)
	 * The texture should be scaled to fit width, with letterboxing top/bottom.
	 * Scale factor: 1600/1920 = 0.833
	 * Rendered size: 1600 x 900
	 * Letterbox: (1200-900)/2 = 150 pixels top and bottom
	 */
	calc_dest_rect(1600, 1200, 1920, 1080, &dest);
	printf("dest_rect: (%.1f, %.1f, %.1f, %.1f)\n", dest.x, dest.y, dest.w, dest.h);

	/* Click in center of window = center of texture */
	window_to_texture_coords(800.0f, 600.0f, &dest, 1920, 1080, &mx, &my);
	check("center click", mx, my, 960, 540);

	/* Click at top-left of rendered area */
	window_to_texture_coords(dest.x, dest.y, &dest, 1920, 1080, &mx, &my);
	check("top-left of texture", mx, my, 0, 0);

	/* Click in top letterbox - should clamp to top edge */
	window_to_texture_coords(800.0f, 0.0f, &dest, 1920, 1080, &mx, &my);
	check("click in top letterbox (clamped)", mx, my, 960, 0);

	/* Click in bottom letterbox - should clamp to bottom edge */
	window_to_texture_coords(800.0f, 1199.0f, &dest, 1920, 1080, &mx, &my);
	check("click in bottom letterbox (clamped)", mx, my, 960, 1079);
}

/*
 * Test: Edge case - zero/invalid dimensions
 */
static void
test_edge_cases(void)
{
	FRect dest;
	int mx, my;

	printf("\n=== Test: Edge cases ===\n");

	/* Zero window - should use fallback */
	calc_dest_rect(0, 768, 1024, 768, &dest);
	window_to_texture_coords(512.0f, 384.0f, &dest, 1024, 768, &mx, &my);
	check("zero width fallback", mx, my, 512, 384);

	/* Zero texture - should handle gracefully */
	calc_dest_rect(1024, 768, 0, 0, &dest);
	/* With zero texture, dest.w and dest.h will be 0, so coords pass through */
	window_to_texture_coords(512.0f, 384.0f, &dest, 0, 0, &mx, &my);
	/* We expect it to not crash - any result is acceptable */
	tests_run++;
	tests_passed++;
	printf("PASS: zero texture dimensions (no crash)\n");
}

/*
 * Test: Fractional coordinates (SDL3 uses float for mouse position)
 */
static void
test_fractional_coords(void)
{
	FRect dest;
	int mx, my;

	printf("\n=== Test: Fractional coordinates ===\n");

	calc_dest_rect(1024, 768, 2048, 1536, &dest);

	/* Sub-pixel mouse position */
	window_to_texture_coords(512.5f, 384.25f, &dest, 2048, 1536, &mx, &my);
	check("sub-pixel position", mx, my, 1025, 768);

	/* Very small fractional part */
	window_to_texture_coords(100.01f, 200.99f, &dest, 2048, 1536, &mx, &my);
	check("small fraction", mx, my, 200, 401);
}

int
main(int argc, char **argv)
{
	(void)argc;
	(void)argv;

	printf("SDL3 Mouse Coordinate Scaling Regression Tests\n");
	printf("(With letterboxing support)\n");
	printf("==============================================\n");

	test_windowed_1to1();
	test_hidpi_2x();
	test_fullscreen_letterbox();
	test_fullscreen_letterbox_vertical();
	test_edge_cases();
	test_fractional_coords();

	printf("\n==============================================\n");
	printf("Results: %d/%d tests passed\n", tests_passed, tests_run);

	return (tests_passed == tests_run) ? 0 : 1;
}
