package io.github.chinmay28.homeapi.core

/**
 * The developer badge thrown up full screen for three seconds, frame by
 * frame: the same timeline as the web GUI's `dev-flash-veil` and
 * `dev-flash-badge` keyframes (frontend/src/App.css), so a tap on the badge
 * looks the same in the app's own screens as on the page.
 *
 *   veil:  fades in over the first 7%, holds, fades out from 82%
 *   badge: lands at 82% size with a small overshoot (102% at 14%, 100% at
 *          22%), holds, then drifts out to 106% as the veil clears
 */
object DevFlash {
    const val DURATION_MS = 3_000L

    /** Where the timeline is at [t] (0 to 1): the veil's opacity and the badge's scale. */
    data class Frame(val veil: Float, val scale: Float)

    private val veil = listOf(0f to 0f, 0.07f to 1f, 0.82f to 1f, 1f to 0f)
    private val badge = listOf(0f to 0.82f, 0.14f to 1.02f, 0.22f to 1f, 0.82f to 1f, 1f to 1.06f)

    fun at(t: Float): Frame {
        val x = t.coerceIn(0f, 1f)
        return Frame(interpolate(veil, x), interpolate(badge, x))
    }

    /** Between two keyframes, eased in and out, as CSS eases each segment. */
    private fun interpolate(keys: List<Pair<Float, Float>>, t: Float): Float {
        for (i in 0 until keys.size - 1) {
            val (t0, v0) = keys[i]
            val (t1, v1) = keys[i + 1]
            if (t <= t1) {
                val p = if (t1 > t0) (t - t0) / (t1 - t0) else 1f
                val eased = p * p * (3 - 2 * p)
                return v0 + (v1 - v0) * eased
            }
        }
        return keys.last().second
    }
}
