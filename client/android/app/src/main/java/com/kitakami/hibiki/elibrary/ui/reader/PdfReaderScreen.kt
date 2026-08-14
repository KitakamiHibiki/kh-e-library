package com.kitakami.hibiki.elibrary.ui.reader

import android.graphics.Bitmap
import androidx.compose.foundation.Image
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTransformGestures
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
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowLeft
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.layout.onSizeChanged
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.res.stringResource
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.IntSize
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.KHApplication
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.reader.PdfPageRenderer
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PdfReaderScreen(
    container: AppContainer,
    bookId: Long,
    onBack: () -> Unit,
) {
    val context = LocalContext.current
    val viewModel: PdfReaderViewModel = viewModel {
        val app = this[ViewModelProvider.AndroidViewModelFactory.APPLICATION_KEY] as KHApplication
        PdfReaderViewModel(app.container, context, bookId)
    }
    var showComplete by remember { mutableStateOf(false) }
    var pageDialog by remember { mutableStateOf(false) }
    var pageBitmap by remember { mutableStateOf<Bitmap?>(null) }
    var viewSize by remember { mutableStateOf(IntSize.Zero) }

    // Zoom state.
    var scale by remember { mutableStateOf(1f) }
    var offsetX by remember { mutableStateOf(0f) }
    var offsetY by remember { mutableStateOf(0f) }

    val renderer = remember(viewModel.file?.absolutePath) {
        viewModel.file?.let { PdfPageRenderer(it) }
    }

    DisposableEffect(renderer) {
        onDispose { renderer?.close() }
    }

    // Page count + restore happens in the VM; report count once renderer is open.
    LaunchedEffect(renderer, viewModel.file) {
        if (renderer != null && viewModel.pageCount == 0) {
            viewModel.onPageCountLoaded(renderer.pageCount)
        }
    }

    // Render the current page off the main thread.
    LaunchedEffect(renderer, viewModel.currentPage, viewSize) {
        val r = renderer ?: return@LaunchedEffect
        if (viewSize.width == 0 || viewSize.height == 0) return@LaunchedEffect
        pageBitmap = withContext(Dispatchers.Default) {
            r.renderPage(viewModel.currentPage.coerceIn(0, r.pageCount - 1), viewSize.width, viewSize.height)
        }
    }

    LaunchedEffect(viewModel.reachedEnd) {
        if (viewModel.reachedEnd) showComplete = true
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.book_read)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.reader_back))
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
            viewModel.error != null || renderer == null -> {
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
                    Box(
                        modifier = Modifier
                            .weight(1f)
                            .fillMaxWidth()
                            .onSizeChanged { viewSize = it },
                        contentAlignment = Alignment.Center,
                    ) {
                        pageBitmap?.let { bmp ->
                            Image(
                                bitmap = bmp.asImageBitmap(),
                                contentDescription = stringResource(R.string.book_read),
                                contentScale = ContentScale.Fit,
                                modifier = Modifier
                                    .fillMaxSize()
                                    .pointerInput(Unit) {
                                        detectTransformGestures { _, pan, zoom, _ ->
                                            scale = (scale * zoom).coerceIn(1f, 5f)
                                            if (scale > 1f) {
                                                offsetX = (offsetX + pan.x).coerceIn(-2000f, 2000f)
                                                offsetY = (offsetY + pan.y).coerceIn(-2000f, 2000f)
                                            } else {
                                                offsetX = 0f
                                                offsetY = 0f
                                            }
                                        }
                                    }
                                    .graphicsLayer {
                                        scaleX = scale
                                        scaleY = scale
                                        translationX = offsetX
                                        translationY = offsetY
                                    },
                            )
                        }
                    }
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 8.dp, vertical = 4.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.SpaceBetween,
                    ) {
                        IconButton(
                            onClick = { viewModel.onPageChanged((viewModel.currentPage - 1).coerceAtLeast(0)) },
                            enabled = viewModel.currentPage > 0,
                        ) {
                            Icon(
                                Icons.AutoMirrored.Filled.KeyboardArrowLeft,
                                contentDescription = stringResource(R.string.reader_epub_prev_page),
                            )
                        }
                        Text(
                            text = stringResource(
                                R.string.reader_page,
                                (viewModel.currentPage + 1).coerceAtLeast(1),
                                viewModel.pageCount.coerceAtLeast(1),
                            ),
                            style = MaterialTheme.typography.bodyMedium,
                            modifier = Modifier
                                .padding(horizontal = 8.dp)
                                .then(Modifier.clickableNoRipple(onClick = { pageDialog = true })),
                        )
                        IconButton(
                            onClick = { viewModel.onPageChanged((viewModel.currentPage + 1).coerceAtMost(viewModel.pageCount - 1)) },
                            enabled = viewModel.currentPage < viewModel.pageCount - 1,
                        ) {
                            Icon(
                                Icons.AutoMirrored.Filled.KeyboardArrowRight,
                                contentDescription = stringResource(R.string.reader_epub_next_page),
                            )
                        }
                    }
                }
            }
        }
    }

    if (showComplete && viewModel.pageCount > 0) {
        ReadCompleteOverlay(
            onMarkFinished = { viewModel.markFinished(onDone = { showComplete = false; onBack() }) },
            onReadAgain = { viewModel.reset(onDone = { showComplete = false }) },
            onBackToLast = { showComplete = false },
        )
    }

    if (pageDialog) {
        var input by remember { mutableStateOf("") }
        AlertDialog(
            onDismissRequest = { pageDialog = false },
            title = { Text(stringResource(R.string.reader_goto_page)) },
            text = {
                OutlinedTextField(
                    value = input,
                    onValueChange = { input = it.filter(Char::isDigit) },
                    singleLine = true,
                )
            },
            confirmButton = {
                TextButton(onClick = {
                    input.toIntOrNull()?.let { viewModel.goToPage(it) }
                    pageDialog = false
                }) {
                    Text(stringResource(R.string.common_confirm))
                }
            },
            dismissButton = {
                TextButton(onClick = { pageDialog = false }) {
                    Text(stringResource(R.string.common_cancel))
                }
            },
        )
    }
}

private fun Modifier.clickableNoRipple(onClick: () -> Unit): Modifier =
    this.then(clickable(onClick = onClick))
