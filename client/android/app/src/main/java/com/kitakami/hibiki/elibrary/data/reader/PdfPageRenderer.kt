package com.kitakami.hibiki.elibrary.data.reader

import android.graphics.Bitmap
import android.graphics.Matrix
import android.graphics.pdf.PdfRenderer
import android.os.ParcelFileDescriptor
import java.io.File

/**
 * Thin wrapper around the framework [PdfRenderer]. Kept on a background thread;
 * renders one page at a time at a fit-width resolution to keep memory bounded.
 */
class PdfPageRenderer(file: File) : AutoCloseable {

    private val fd = ParcelFileDescriptor.open(file, ParcelFileDescriptor.MODE_READ_ONLY)
    private val renderer = PdfRenderer(fd)

    val pageCount: Int get() = renderer.pageCount

    /** Renders page [index] scaled to fit [targetWidth]x[targetHeight], preserving aspect. */
    fun renderPage(index: Int, targetWidth: Int, targetHeight: Int): Bitmap {
        val page = renderer.openPage(index)
        try {
            val scale = (targetWidth.toFloat() / page.width).coerceAtMost(targetHeight.toFloat() / page.height)
            val w = (page.width * scale).toInt().coerceAtLeast(1)
            val h = (page.height * scale).toInt().coerceAtLeast(1)
            val bitmap = Bitmap.createBitmap(w, h, Bitmap.Config.ARGB_8888)
            val matrix = Matrix().apply {
                if (w != page.width || h != page.height) {
                    setScale(w.toFloat() / page.width, h.toFloat() / page.height)
                }
            }
            page.render(bitmap, null, matrix, PdfRenderer.Page.RENDER_MODE_FOR_DISPLAY)
            return bitmap
        } finally {
            page.close()
        }
    }

    override fun close() {
        renderer.close()
        fd.close()
    }
}
