package io.github.chinmay28.homeapi

import android.animation.ValueAnimator
import android.app.Activity
import android.graphics.Typeface
import android.util.TypedValue
import android.view.Gravity
import android.view.View
import android.view.ViewGroup
import android.view.animation.LinearInterpolator
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.TextView
import androidx.core.graphics.ColorUtils
import com.google.android.material.imageview.ShapeableImageView
import com.google.android.material.shape.RelativeCornerSize
import com.google.android.material.shape.ShapeAppearanceModel
import io.github.chinmay28.homeapi.core.DevFlash
import kotlin.math.min

/**
 * The developer badge, done natively: the same mark the web GUI carries in
 * its header, thrown up full screen for three seconds when tapped — the badge
 * over `github.com/chinmay28` in monospace, on a veil that fades in and out,
 * landing with a small overshoot ([DevFlash] is the timeline). A tap
 * dismisses it early.
 */
object DevBadge {
    /** Rounds an image view into the badge's circle. */
    fun circle(view: ShapeableImageView) {
        view.shapeAppearanceModel = ShapeAppearanceModel.builder().setAllCornerSizes(RelativeCornerSize(0.5f)).build()
    }

    fun flash(activity: Activity) {
        val root = activity.findViewById<ViewGroup>(android.R.id.content) ?: return
        if (root.findViewWithTag<View>(TAG) != null) return
        val dm = activity.resources.displayMetrics
        fun dp(v: Float) = TypedValue.applyDimension(TypedValue.COMPLEX_UNIT_DIP, v, dm)

        // min(64vmin, 26rem), as on the page.
        val size = min(min(dm.widthPixels, dm.heightPixels) * 0.64f, dp(416f)).toInt()
        val badge = ShapeableImageView(activity).apply {
            setImageResource(R.drawable.dev_badge_full)
            contentDescription = activity.getString(R.string.dev_badge_full)
            circle(this)
            elevation = dp(12f)
        }
        val handle = TextView(activity).apply {
            text = activity.getString(R.string.dev_handle)
            typeface = Typeface.MONOSPACE
            letterSpacing = 0.08f
            setTextColor(activity.getColor(R.color.homeapi_muted))
            setTextSize(TypedValue.COMPLEX_UNIT_SP, 15f)
        }
        val lockup = LinearLayout(activity).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER_HORIZONTAL
            addView(badge, LinearLayout.LayoutParams(size, size))
            addView(handle, LinearLayout.LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT).apply {
                topMargin = dp(21.6f).toInt()
            })
        }
        val veil = FrameLayout(activity).apply {
            tag = TAG
            // The page's veil: the background at 78%.
            setBackgroundColor(ColorUtils.setAlphaComponent(activity.getColor(R.color.homeapi_bg), (0.78f * 255).toInt()))
            isClickable = true
            addView(lockup, FrameLayout.LayoutParams(ViewGroup.LayoutParams.WRAP_CONTENT, ViewGroup.LayoutParams.WRAP_CONTENT, Gravity.CENTER))
        }

        var animator: ValueAnimator? = null
        val dismiss = Runnable {
            animator?.cancel()
            root.removeView(veil)
        }
        veil.setOnClickListener { dismiss.run() }
        root.addView(veil, ViewGroup.LayoutParams(ViewGroup.LayoutParams.MATCH_PARENT, ViewGroup.LayoutParams.MATCH_PARENT))

        if (!ValueAnimator.areAnimatorsEnabled()) {
            // Animations off in the system settings: show it still, for the
            // same three seconds.
            veil.postDelayed(dismiss, DevFlash.DURATION_MS)
            return
        }
        animator = ValueAnimator.ofFloat(0f, 1f).apply {
            duration = DevFlash.DURATION_MS
            interpolator = LinearInterpolator()
            addUpdateListener {
                val f = DevFlash.at(it.animatedValue as Float)
                veil.alpha = f.veil
                lockup.scaleX = f.scale
                lockup.scaleY = f.scale
            }
            doOnEnd { root.removeView(veil) }
            start()
        }
    }

    private fun ValueAnimator.doOnEnd(block: () -> Unit) = addListener(object : android.animation.AnimatorListenerAdapter() {
        override fun onAnimationEnd(animation: android.animation.Animator) = block()
    })

    private const val TAG = "homeapi-dev-flash"
}
