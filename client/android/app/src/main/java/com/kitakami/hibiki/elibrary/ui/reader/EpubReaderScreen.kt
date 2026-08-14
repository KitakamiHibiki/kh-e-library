package com.kitakami.hibiki.elibrary.ui.reader

import android.annotation.SuppressLint
import android.net.Uri
import android.webkit.JavascriptInterface
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.DarkMode
import androidx.compose.material.icons.filled.FormatSize
import androidx.compose.material.icons.filled.List
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Slider
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.KHApplication
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.ui.theme.BrandBlue

@SuppressLint("SetJavaScriptEnabled")
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun EpubReaderScreen(
    container: AppContainer,
    bookId: Long,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val viewModel: EpubReaderViewModel = viewModel {
        val app = this[ViewModelProvider.AndroidViewModelFactory.APPLICATION_KEY] as KHApplication
        EpubReaderViewModel(app.container, context, bookId)
    }
    val webViewRef = remember { mutableStateOf<WebView?>(null) }
    var showToc by remember { mutableStateOf(false) }
    var showFontDialog by remember { mutableStateOf(false) }
    var showComplete by remember { mutableStateOf(false) }
    val chapter = viewModel.currentChapter()

    // Inject styles + restore scroll after each page load.
    LaunchedEffect(chapter?.index, viewModel.fontSize, viewModel.darkMode) {
        val wv = webViewRef.value ?: return@LaunchedEffect
        val ch = viewModel.currentChapter() ?: return@LaunchedEffect
        wv.loadUrl(Uri.fromFile(ch.file).toString())
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(chapter?.title ?: stringResource(R.string.book_read), maxLines = 1) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.reader_back))
                    }
                },
                actions = {
                    IconButton(onClick = { showToc = true }) {
                        Icon(Icons.Default.List, contentDescription = stringResource(R.string.reader_toc))
                    }
                    IconButton(onClick = { showFontDialog = true }) {
                        Icon(Icons.Default.FormatSize, contentDescription = stringResource(R.string.reader_font_size))
                    }
                    IconButton(onClick = { viewModel.toggleDark() }) {
                        Icon(
                            Icons.Default.DarkMode,
                            contentDescription = null,
                            tint = if (viewModel.darkMode) BrandBlue else MaterialTheme.colorScheme.onSurfaceVariant,
                        )
                    }
                },
            )
        },
    ) { innerPadding ->
        when {
            viewModel.loading -> {
                Box(Modifier.fillMaxSize().padding(innerPadding), contentAlignment = Alignment.Center) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        CircularProgressIndicator()
                        Spacer(Modifier.height(8.dp))
                        Text(stringResource(R.string.reader_downloading))
                    }
                }
            }
            viewModel.error != null -> {
                Column(
                    modifier = Modifier.fillMaxSize().padding(innerPadding),
                    verticalArrangement = Arrangement.Center,
                    horizontalAlignment = Alignment.CenterHorizontally,
                ) {
                    Text(
                        stringResource(R.string.reader_open_failed, viewModel.error ?: ""),
                        color = MaterialTheme.colorScheme.error,
                    )
                    Button(onClick = onBack, modifier = Modifier.padding(top = 12.dp)) {
                        Text(stringResource(R.string.reader_back))
                    }
                }
            }
            else -> {
                Column(Modifier.fillMaxSize().padding(innerPadding)) {
                    Box(Modifier.weight(1f).fillMaxWidth()) {
                        AndroidView(
                            factory = { ctx ->
                                WebView(ctx).apply {
                                    settings.javaScriptEnabled = true
                                    settings.allowFileAccess = true
                                    settings.domStorageEnabled = true
                                    @Suppress("DEPRECATION")
                                    settings.allowUniversalAccessFromFileURLs = true
                                    @Suppress("DEPRECATION")
                                    settings.allowFileAccessFromFileURLs = true
                                    webViewClient = object : WebViewClient() {
                                        override fun onPageFinished(view: WebView?, url: String?) {
                                            injectReaderStyles(view, viewModel.fontSize, viewModel.darkMode)
                                            restoreScroll(view, viewModel.ratio)
                                        }
                                    }
                                    addJavascriptInterface(ReaderBridge(viewModel), "ReaderBridge")
                                    webViewRef.value = this
                                }
                            },
                        )
                    }
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 8.dp, vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        IconButton(onClick = { viewModel.prev() }) {
                            Icon(
                                Icons.AutoMirrored.Filled.KeyboardArrowLeft,
                                contentDescription = stringResource(R.string.reader_prev_chapter),
                            )
                        }
                        Text(
                            text = stringResource(
                                R.string.reader_page,
                                (viewModel.chapterIndex + 1),
                                (viewModel.book?.chapters?.size ?: 1),
                            ),
                            style = MaterialTheme.typography.bodyMedium,
                        )
                        IconButton(onClick = {
                            val moved = viewModel.next()
                            if (!moved) showComplete = true
                        }) {
                            Icon(
                                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                                contentDescription = stringResource(R.string.reader_next_chapter),
                            )
                        }
                    }
                }
            }
        }
    }

    // TOC dialog.
    if (showToc) {
        val chapters = viewModel.book?.chapters.orEmpty()
        AlertDialog(
            onDismissRequest = { showToc = false },
            title = { Text(stringResource(R.string.reader_toc)) },
            text = {
                LazyColumn {
                    items(chapters) { ch ->
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier
                                .fillMaxWidth()
                                .clickable {
                                    viewModel.goToChapter(ch.index)
                                    showToc = false
                                }
                                .padding(vertical = 10.dp),
                        ) {
                            Text(
                                text = ch.title,
                                style = MaterialTheme.typography.bodyMedium,
                                color = if (ch.index == viewModel.chapterIndex) MaterialTheme.colorScheme.primary
                                else MaterialTheme.colorScheme.onSurface,
                                modifier = Modifier.weight(1f),
                            )
                        }
                    }
                }
            },
            confirmButton = {
                TextButton(onClick = { showToc = false }) {
                    Text(stringResource(R.string.common_cancel))
                }
            },
        )
    }

    // Font size dialog.
    if (showFontDialog) {
        AlertDialog(
            onDismissRequest = { showFontDialog = false },
            title = { Text(stringResource(R.string.reader_font_size)) },
            text = {
                Column {
                    Text("${viewModel.fontSize}px")
                    Slider(
                        value = viewModel.fontSize.toFloat(),
                        onValueChange = { viewModel.updateFontSize(it.toInt()) },
                        valueRange = 12f..32f,
                        steps = 19,
                    )
                }
            },
            confirmButton = {
                TextButton(onClick = { showFontDialog = false }) {
                    Text(stringResource(R.string.common_confirm))
                }
            },
        )
    }

    // Completion page.
    if (showComplete) {
        ReadCompleteOverlay(
            onMarkFinished = { viewModel.markFinished(onDone = { showComplete = false; onBack() }) },
            onReadAgain = { viewModel.readAgain(onDone = { showComplete = false }) },
            onBackToLast = { viewModel.backToLastPage(); showComplete = false },
        )
    }
}

/** Bridge called from JS on scroll. Must hold a stable VM reference. */
private class ReaderBridge(private val viewModel: EpubReaderViewModel) {
    @JavascriptInterface
    fun onScroll(ratio: Float) {
        viewModel.onScrollRatio(ratio)
    }
}

private fun injectReaderStyles(webView: WebView?, fontSize: Int, dark: Boolean) {
    webView ?: return
    val bg = if (dark) "#121318" else "#ffffff"
    val fg = if (dark) "#c6c5d0" else "#1b1b21"
    val js = """
        (function() {
            var style = document.createElement('style');
            style.textContent = 'html, body { background: $bg !important; color: $fg !important; } ' +
                'body { font-size: ${fontSize}px !important; line-height: 1.7 !important; } ' +
                'p, div, span, li, td { color: $fg !important; } ' +
                'a { color: #4c6fff !important; }';
            document.head.appendChild(style);
            if (!window.__readerScrollAttached) {
                window.__readerScrollAttached = true;
                window.addEventListener('scroll', function() {
                    var h = document.documentElement;
                    var max = h.scrollHeight - h.clientHeight;
                    var r = max > 0 ? (h.scrollTop / max) : 0;
                    window.ReaderBridge.onScroll(r);
                });
            }
        })();
    """.trimIndent()
    webView.evaluateJavascript(js, null)
}

private fun restoreScroll(webView: WebView?, ratio: Float) {
    webView ?: return
    val js = """
        (function() {
            var h = document.documentElement;
            var max = h.scrollHeight - h.clientHeight;
            if (max > 0) h.scrollTop = max * ${ratio.coerceIn(0f, 1f)};
        })();
    """.trimIndent()
    webView.evaluateJavascript(js, null)
}
