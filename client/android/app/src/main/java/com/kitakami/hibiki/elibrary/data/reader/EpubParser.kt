package com.kitakami.hibiki.elibrary.data.reader

import org.w3c.dom.Element
import java.io.File
import java.util.zip.ZipEntry
import java.util.zip.ZipFile
import javax.xml.parsers.DocumentBuilderFactory

/** One spine chapter of an EPUB, ready for WebView rendering. */
data class EpubChapter(
    val index: Int,
    val href: String,
    val title: String,
    val file: File,
)

/** Parsed EPUB: root dir + ordered chapters (reading order = spine). */
data class EpubBook(
    val title: String,
    val chapters: List<EpubChapter>,
    val rootDir: File,
)

/**
 * Minimal EPUB 2/3 parser. No external dependencies: unzips the archive, reads
 * `META-INF/container.xml` → OPF path, walks the spine+manifest to build the
 * reading order, and (best-effort) pulls titles from the NCX navMap.
 */
object EpubParser {

    fun parse(epubFile: File, extractDir: File): EpubBook {
        extractDir.mkdirs()
        ZipFile(epubFile).use { zip -> extractZip(zip, extractDir) }

        val containerXml = File(extractDir, "META-INF/container.xml")
        val opfPath = if (containerXml.exists()) {
            val doc = parseXml(containerXml)
            val rootfile = doc.getElementsByTagName("rootfile")
            if (rootfile.length == 0) throw EpubParseException("container.xml: no rootfile")
            (rootfile.item(0) as Element).getAttribute("full-path")
                .takeIf { it.isNotBlank() }
                ?: throw EpubParseException("container.xml: empty rootfile path")
        } else {
            throw EpubParseException("EPUB 缺少 META-INF/container.xml")
        }

        val opfFile = File(extractDir, opfPath)
        if (!opfFile.exists()) throw EpubParseException("找不到 OPF: $opfPath")
        val opfDir = opfFile.parentFile ?: extractDir
        val opf = parseXml(opfFile)

        // Title from metadata.
        val title = getElementText(opf, "title") ?: "未知书名"

        // Manifest: id -> href.
        val manifestHrefs = mutableMapOf<String, String>()
        val manifestNodes = opf.getElementsByTagName("item")
        for (i in 0 until manifestNodes.length) {
            val item = manifestNodes.item(i) as Element
            val id = item.getAttribute("id")
            val href = item.getAttribute("href")
            if (id.isNotBlank() && href.isNotBlank()) manifestHrefs[id] = href
        }

        // Spine itemrefs → chapter hrefs in reading order.
        val chapterHrefs = mutableListOf<String>()
        val spineNodes = opf.getElementsByTagName("itemref")
        for (i in 0 until spineNodes.length) {
            val ref = spineNodes.item(i) as Element
            val idref = ref.getAttribute("idref")
            val href = manifestHrefs[idref]
            if (href != null && href !in chapterHrefs) chapterHrefs.add(href)
        }
        if (chapterHrefs.isEmpty()) throw EpubParseException("EPUB 没有任何章节（spine 为空）")

        // Best-effort chapter titles from NCX.
        val tocTitles = parseNcxTitles(extractDir, opfDir, opf, manifestHrefs)

        val chapters = chapterHrefs.mapIndexed { index, href ->
            val file = File(opfDir, href)
            val title = tocTitles[href] ?: "第 ${index + 1} 章"
            EpubChapter(index, href, title, file)
        }

        return EpubBook(title = title, chapters = chapters, rootDir = extractDir)
    }

    private fun parseNcxTitles(
        extractDir: File,
        opfDir: File,
        opf: org.w3c.dom.Document,
        manifestHrefs: Map<String, String>,
    ): Map<String, String> {
        // Prefer nav.xhtml (EPUB3) <a href> inside <nav epub:type="toc">.
        val tocHref = opf.getElementsByTagName("item")
            .let { nodes ->
                for (i in 0 until nodes.length) {
                    val it = nodes.item(i) as Element
                    val media = it.getAttribute("media-type")
                    val props = it.getAttribute("properties")
                    if (media.contains("xhtml") && (props.contains("nav") || it.getAttribute("id").contains("nav"))) {
                        val href = it.getAttribute("href")
                        if (href.isNotBlank()) return@let href
                    }
                }
                null
            }
        val result = mutableMapOf<String, String>()
        if (tocHref != null) {
            val navFile = File(opfDir, tocHref)
            if (navFile.exists()) {
                try {
                    val doc = parseXml(navFile)
                    val links = doc.getElementsByTagName("a")
                    for (i in 0 until links.length) {
                        val a = links.item(i) as Element
                        val href = a.getAttribute("href").substringBefore('#')
                        val text = a.textContent.trim()
                        if (href.isNotBlank() && text.isNotBlank()) result[href] = text
                    }
                } catch (_: Exception) {
                }
            }
        }

        // Fallback: NCX toc.ncx navPoint/navLabel.
        if (result.isEmpty()) {
            val ncxPath = opf.getElementsByTagName("item")
                .let { nodes ->
                    for (i in 0 until nodes.length) {
                        val it = nodes.item(i) as Element
                        if (it.getAttribute("media-type").contains("ncx")) return@let it.getAttribute("href")
                    }
                    null
                }
            if (ncxPath != null) {
                val ncxFile = File(opfDir, ncxPath)
                if (ncxFile.exists()) {
                    try {
                        val doc = parseXml(ncxFile)
                        val navPoints = doc.getElementsByTagName("navPoint")
                        for (i in 0 until navPoints.length) {
                            val np = navPoints.item(i) as Element
                            val label = getElementTextFrom(np, "text")
                            val content = getFirstChildElement(np, "content")
                            val src = content?.getAttribute("src")
                            if (label != null && src != null) {
                                result[src.substringBefore('#')] = label
                            }
                        }
                    } catch (_: Exception) {
                    }
                }
            }
        }
        return result
    }

    private fun extractZip(zip: ZipFile, dest: File) {
        val entries = zip.entries()
        while (entries.hasMoreElements()) {
            val entry: ZipEntry = entries.nextElement()
            val safeName = entry.name.replace('\\', '/').trimStart('/')
            val outFile = File(dest, safeName)
            // Guard against zip-slip.
            if (!outFile.canonicalPath.startsWith(dest.canonicalPath)) continue
            if (entry.isDirectory) {
                outFile.mkdirs()
                continue
            }
            outFile.parentFile?.mkdirs()
            zip.getInputStream(entry).use { input ->
                outFile.outputStream().use { output -> input.copyTo(output) }
            }
        }
    }

    private fun parseXml(file: File): org.w3c.dom.Document {
        val factory = DocumentBuilderFactory.newInstance()
        factory.isNamespaceAware = true
        // Disable external entities / DTD fetching (XXE defense). These URIs are
        // not recognized by every parser — notably Android's built-in kxml2 DOM
        // throws on the Apache Xerces feature — so each is set defensively.
        setFeatureQuietly(factory, "http://xml.org/sax/features/external-general-entities", false)
        setFeatureQuietly(factory, "http://xml.org/sax/features/external-parameter-entities", false)
        setFeatureQuietly(factory, "http://apache.org/xml/features/nonvalidating/load-external-dtd", false)
        return factory.newDocumentBuilder().parse(file)
    }

    private fun setFeatureQuietly(factory: DocumentBuilderFactory, feature: String, value: Boolean) {
        try {
            factory.setFeature(feature, value)
        } catch (_: Exception) {
            // Feature unrecognized by this parser implementation — safe to skip.
        }
    }

    private fun getElementText(doc: org.w3c.dom.Document, tag: String): String? {
        val nodes = doc.getElementsByTagName(tag)
        return if (nodes.length > 0) nodes.item(0).textContent.trim() else null
    }

    private fun getElementTextFrom(el: Element, tag: String): String? {
        val nodes = el.getElementsByTagName(tag)
        return if (nodes.length > 0) nodes.item(0).textContent.trim() else null
    }

    private fun getFirstChildElement(el: Element, tag: String): Element? {
        val nodes = el.getElementsByTagName(tag)
        return if (nodes.length > 0) nodes.item(0) as? Element else null
    }
}

class EpubParseException(message: String) : Exception(message)
