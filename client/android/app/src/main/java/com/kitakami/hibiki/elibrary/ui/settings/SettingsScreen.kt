package com.kitakami.hibiki.elibrary.ui.settings

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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Dns
import androidx.compose.material.icons.filled.Label
import androidx.compose.material.icons.filled.Leaderboard
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
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
import androidx.lifecycle.ViewModelProvider
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.lifecycle.viewmodel.compose.viewModel
import com.kitakami.hibiki.elibrary.AppContainer
import com.kitakami.hibiki.elibrary.KHApplication
import com.kitakami.hibiki.elibrary.R
import com.kitakami.hibiki.elibrary.data.model.StatsOverview
import com.kitakami.hibiki.elibrary.ui.theme.BrandGreen
import com.kitakami.hibiki.elibrary.ui.theme.BrandRed

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    container: AppContainer,
    onBack: () -> Unit,
    onOpenServerConfig: () -> Unit,
    onOpenTags: () -> Unit,
) {
    val viewModel: SettingsViewModel = viewModel {
        val app = this[ViewModelProvider.AndroidViewModelFactory.APPLICATION_KEY] as KHApplication
        SettingsViewModel(app.container)
    }
    val context = LocalContext.current
    val snackbarHostState = remember { SnackbarHostState() }
    var installConfirm by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) { viewModel.load() }

    LaunchedEffect(viewModel.message) {
        val msg = viewModel.message ?: return@LaunchedEffect
        snackbarHostState.showSnackbar(context.getString(msg.resId, *msg.args.toTypedArray()))
        viewModel.consumeMessage()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text(stringResource(R.string.settings_title)) },
                navigationIcon = {
                    IconButton(onClick = onBack) {
                        Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = stringResource(R.string.reader_back))
                    }
                },
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .verticalScroll(rememberScrollState())
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            // Entry cards.
            EntryCard(
                icon = { Icon(Icons.Default.Dns, null) },
                label = stringResource(R.string.settings_server_config),
                onClick = onOpenServerConfig,
            )
            EntryCard(
                icon = { Icon(Icons.Default.Label, null) },
                label = stringResource(R.string.settings_tags),
                onClick = onOpenTags,
            )

            // Stats overview.
            Card {
                Column(Modifier.padding(16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.Leaderboard, null)
                        Spacer(Modifier.width(8.dp))
                        Text(stringResource(R.string.settings_stats), style = MaterialTheme.typography.titleMedium)
                    }
                    Spacer(Modifier.height(12.dp))
                    val stats = viewModel.stats
                    if (stats == null) {
                        Text(stringResource(R.string.common_loading), color = MaterialTheme.colorScheme.onSurfaceVariant)
                    } else {
                        StatsRow(stats)
                    }
                }
            }

            // Runtime (server-shared) settings.
            Card {
                Column(Modifier.padding(16.dp)) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Icon(Icons.Default.Settings, null)
                        Spacer(Modifier.width(8.dp))
                        Text(stringResource(R.string.settings_runtime), style = MaterialTheme.typography.titleMedium)
                    }
                    Spacer(Modifier.height(8.dp))
                    RuntimeSettingsEditor(
                        settings = viewModel.settings,
                        onSave = viewModel::save,
                        saving = viewModel.saving,
                    )
                }
            }

            // Software update.
            Card {
                Column(Modifier.padding(16.dp)) {
                    Text(stringResource(R.string.settings_update), style = MaterialTheme.typography.titleMedium)
                    Spacer(Modifier.height(12.dp))
                    UpdateSection(
                        viewModel = viewModel,
                        onInstall = { installConfirm = true },
                    )
                }
            }
        }
    }

    viewModel.updateResult?.let { result ->
        if (installConfirm && result.has_update) {
            AlertDialog(
                onDismissRequest = { installConfirm = false },
                title = { Text(stringResource(R.string.update_install)) },
                text = { Text(stringResource(R.string.update_install_confirm)) },
                confirmButton = {
                    TextButton(onClick = {
                        installConfirm = false
                        viewModel.startUpdate(result.download_url)
                    }) {
                        Text(stringResource(R.string.common_confirm))
                    }
                },
                dismissButton = {
                    TextButton(onClick = { installConfirm = false }) {
                        Text(stringResource(R.string.common_cancel))
                    }
                },
            )
        }
    }
}

@Composable
private fun EntryCard(
    icon: @Composable () -> Unit,
    label: String,
    onClick: () -> Unit,
) {
    Card(onClick = onClick) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
        ) {
            icon()
            Spacer(Modifier.width(12.dp))
            Text(label, style = MaterialTheme.typography.bodyLarge)
        }
    }
}

@Composable
private fun StatsRow(stats: StatsOverview) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
    ) {
        StatCell(stringResource(R.string.stats_total), stats.total_books)
        StatCell(stringResource(R.string.status_unread), stats.unread_count)
        StatCell(stringResource(R.string.status_reading), stats.reading_count)
        StatCell(stringResource(R.string.status_finished), stats.finished_count, color = BrandGreen)
    }
    Spacer(Modifier.height(12.dp))
    Text(stringResource(R.string.stats_recent), style = MaterialTheme.typography.labelLarge)
    Spacer(Modifier.height(4.dp))
    if (stats.recent_reading.isEmpty()) {
        Text(stringResource(R.string.stats_recent_empty), color = MaterialTheme.colorScheme.onSurfaceVariant)
    } else {
        stats.recent_reading.take(5).forEach { item ->
            Text(
                text = item.title,
                style = MaterialTheme.typography.bodyMedium,
                modifier = Modifier.padding(vertical = 2.dp),
                maxLines = 1,
            )
        }
    }
}

@Composable
private fun StatCell(label: String, value: Int, color: androidx.compose.ui.graphics.Color? = null) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(
            text = value.toString(),
            style = MaterialTheme.typography.titleLarge,
            color = color ?: MaterialTheme.colorScheme.primary,
        )
        Text(
            text = label,
            style = MaterialTheme.typography.labelMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )
    }
}

@Composable
private fun RuntimeSettingsEditor(
    settings: Map<String, String>,
    onSave: (Map<String, String>) -> Unit,
    saving: Boolean,
) {
    val pageSize = remember(settings) { mutableStateOf(settings["ui.page_size"] ?: "20") }
    val fontSize = remember(settings) { mutableStateOf(settings["reader.font_size"] ?: "16") }
    val theme = remember(settings) { mutableStateOf(settings["ui.theme"] ?: "light") }
    val sortField = remember(settings) { mutableStateOf(settings["ui.sort_field"] ?: "updated_at") }
    val sortOrder = remember(settings) { mutableStateOf(settings["ui.sort_order"] ?: "desc") }
    val pdfView = remember(settings) { mutableStateOf(settings["reader.pdf_view_mode"] ?: "single") }
    val epubView = remember(settings) { mutableStateOf(settings["reader.epub_view_mode"] ?: "single") }

    DropdownSetting(
        label = stringResource(R.string.setting_theme),
        value = theme.value,
        options = listOf(
            "light" to stringResource(R.string.setting_theme_light),
            "dark" to stringResource(R.string.setting_theme_dark),
            "auto" to stringResource(R.string.setting_theme_auto),
        ),
        onSelect = { theme.value = it },
    )
    DropdownSetting(
        label = stringResource(R.string.setting_sort_field),
        value = sortField.value,
        options = listOf(
            "updated_at" to stringResource(R.string.bookshelf_sort_updated_at),
            "title" to stringResource(R.string.bookshelf_sort_title),
            "author" to stringResource(R.string.bookshelf_sort_author),
            "created_at" to stringResource(R.string.bookshelf_sort_created_at),
        ),
        onSelect = { sortField.value = it },
    )
    DropdownSetting(
        label = stringResource(R.string.setting_sort_order),
        value = sortOrder.value,
        options = listOf(
            "asc" to stringResource(R.string.bookshelf_sort_asc),
            "desc" to stringResource(R.string.bookshelf_sort_desc),
        ),
        onSelect = { sortOrder.value = it },
    )
    DropdownSetting(
        label = stringResource(R.string.setting_pdf_view_mode),
        value = pdfView.value,
        options = listOf(
            "single" to stringResource(R.string.setting_view_single),
            "double" to stringResource(R.string.setting_view_double),
            "scroll" to stringResource(R.string.setting_view_scroll),
        ),
        onSelect = { pdfView.value = it },
    )
    DropdownSetting(
        label = stringResource(R.string.setting_epub_view_mode),
        value = epubView.value,
        options = listOf(
            "single" to stringResource(R.string.setting_view_single),
            "double" to stringResource(R.string.setting_view_double),
        ),
        onSelect = { epubView.value = it },
    )
    NumberSetting(stringResource(R.string.setting_page_size), pageSize.value) { pageSize.value = it }
    NumberSetting(stringResource(R.string.setting_font_size), fontSize.value) { fontSize.value = it }

    Spacer(Modifier.height(8.dp))
    Button(
        onClick = {
            onSave(
                mapOf(
                    "ui.theme" to theme.value,
                    "ui.sort_field" to sortField.value,
                    "ui.sort_order" to sortOrder.value,
                    "reader.pdf_view_mode" to pdfView.value,
                    "reader.epub_view_mode" to epubView.value,
                    "ui.page_size" to pageSize.value,
                    "reader.font_size" to fontSize.value,
                ),
            )
        },
        enabled = !saving,
        modifier = Modifier.fillMaxWidth(),
    ) {
        Text(if (saving) stringResource(R.string.common_saving) else stringResource(R.string.setting_save))
    }
}

@Composable
private fun DropdownSetting(
    label: String,
    value: String,
    options: List<Pair<String, String>>,
    onSelect: (String) -> Unit,
) {
    var expanded by remember { mutableStateOf(false) }
    val labelOf = options.firstOrNull { it.first == value }?.second ?: value
    Column(Modifier.padding(vertical = 6.dp)) {
        Text(label, style = MaterialTheme.typography.labelLarge)
        Box {
            OutlinedButton(
                onClick = { expanded = true },
                modifier = Modifier.fillMaxWidth(),
            ) {
                Text(labelOf)
            }
            DropdownMenu(expanded = expanded, onDismissRequest = { expanded = false }) {
                options.forEach { (v, l) ->
                    DropdownMenuItem(
                        text = { Text(l) },
                        onClick = { onSelect(v); expanded = false },
                    )
                }
            }
        }
    }
}

@Composable
private fun NumberSetting(label: String, value: String, onChange: (String) -> Unit) {
    Column(Modifier.padding(vertical = 6.dp)) {
        Text(label, style = MaterialTheme.typography.labelLarge)
        OutlinedTextField(
            value = value,
            onValueChange = { onChange(it.filter(Char::isDigit)) },
            singleLine = true,
            modifier = Modifier.fillMaxWidth(),
        )
    }
}

@Composable
private fun UpdateSection(
    viewModel: SettingsViewModel,
    onInstall: () -> Unit,
) {
    Text(
        text = stringResource(R.string.update_current, viewModel.currentVersion.ifBlank { "-" }),
        style = MaterialTheme.typography.bodyMedium,
        color = MaterialTheme.colorScheme.onSurfaceVariant,
    )
    Spacer(Modifier.height(8.dp))

    if (viewModel.updateRunning) {
        val status = viewModel.updateStatus
        when (status?.state) {
            "downloading" -> {
                Text(stringResource(R.string.update_downloading, status.progress))
                LinearProgressIndicator(
                    progress = { (status.progress / 100f).coerceIn(0f, 1f) },
                    modifier = Modifier.fillMaxWidth().padding(top = 8.dp),
                )
            }
            "installing" -> Text(stringResource(R.string.update_installing))
            "completed" -> Text(stringResource(R.string.update_completed), color = BrandGreen)
            "failed" -> Text(stringResource(R.string.update_failed_state, status.message), color = BrandRed)
            else -> Text(stringResource(R.string.common_loading))
        }
    } else {
        OutlinedButton(onClick = viewModel::checkUpdate, enabled = !viewModel.checkingUpdate) {
            if (viewModel.checkingUpdate) {
                CircularProgressIndicator(Modifier.width(18.dp).height(18.dp), strokeWidth = 2.dp)
                Spacer(Modifier.width(8.dp))
            }
            Text(stringResource(R.string.update_check))
        }

        viewModel.updateResult?.let { result ->
            Spacer(Modifier.height(8.dp))
            Text(
                stringResource(R.string.update_latest, result.latest_version),
                style = MaterialTheme.typography.bodyMedium,
            )
            if (result.has_update) {
                Spacer(Modifier.height(4.dp))
                Text(
                    text = stringResource(R.string.update_release_notes),
                    style = MaterialTheme.typography.labelLarge,
                )
                Text(
                    text = result.release_notes.ifBlank { stringResource(R.string.update_available) },
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                )
                Spacer(Modifier.height(8.dp))
                Button(onClick = onInstall) {
                    Text(stringResource(R.string.update_install))
                }
            } else {
                Spacer(Modifier.height(4.dp))
                Text(stringResource(R.string.update_up_to_date), color = BrandGreen)
            }
        }
    }
}
