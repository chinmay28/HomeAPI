package io.github.chinmay28.homeapi.core

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

class DevFlashTest {
    private fun near(want: Float, got: Float) = assertEquals(want, got, 0.0001f)

    @Test
    fun followsTheWebKeyframes() {
        near(0f, DevFlash.at(0f).veil)
        near(0.82f, DevFlash.at(0f).scale)
        near(1f, DevFlash.at(0.07f).veil)
        near(1.02f, DevFlash.at(0.14f).scale)
        near(1f, DevFlash.at(0.22f).scale)
        near(1f, DevFlash.at(0.5f).veil)
        near(1f, DevFlash.at(0.82f).veil)
        near(0f, DevFlash.at(1f).veil)
        near(1.06f, DevFlash.at(1f).scale)
    }

    @Test
    fun overshootsOnTheWayIn() {
        assertTrue(DevFlash.at(0.17f).scale > 1f)
        assertTrue(DevFlash.at(0.90f).veil in 0f..1f)
    }

    @Test
    fun outOfRangeIsClamped() {
        assertEquals(DevFlash.at(0f), DevFlash.at(-1f))
        assertEquals(DevFlash.at(1f), DevFlash.at(2f))
    }
}
