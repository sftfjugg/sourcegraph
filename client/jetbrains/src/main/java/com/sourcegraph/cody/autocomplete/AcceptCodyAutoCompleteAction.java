package com.sourcegraph.cody.autocomplete;

import com.intellij.openapi.actionSystem.DataContext;
import com.intellij.openapi.application.WriteAction;
import com.intellij.openapi.editor.*;
import com.intellij.openapi.editor.actionSystem.EditorAction;
import com.intellij.openapi.editor.actionSystem.EditorActionHandler;
import com.intellij.openapi.project.Project;
import com.intellij.openapi.util.TextRange;
import com.sourcegraph.cody.agent.CodyAgent;
import com.sourcegraph.cody.agent.CodyAgentServer;
import com.sourcegraph.cody.autocomplete.render.CodyAutoCompleteElementRenderer;
import com.sourcegraph.cody.vscode.InlineAutoCompleteItem;
import com.sourcegraph.common.EditorUtils;
import com.sourcegraph.telemetry.GraphQlLogger;
import java.util.List;
import java.util.Optional;
import org.jetbrains.annotations.NotNull;
import org.jetbrains.annotations.Nullable;

/**
 * The action that gets triggered when the user accepts a Cody completion.
 *
 * <p>The action works by reading the Inlay at the caret position and inserting the completion text
 * into the editor.
 */
public class AcceptCodyAutoCompleteAction extends EditorAction {
  public AcceptCodyAutoCompleteAction() {
    super(new AcceptCompletionActionHandler());
  }

  private static class AcceptCompletionActionHandler extends EditorActionHandler {

    @Override
    protected boolean isEnabledForCaret(
        @NotNull Editor editor, @NotNull Caret caret, DataContext dataContext) {
      // Returns false to fall back to normal TAB character if there is no suggestion at the caret.
      return editor.getProject() != null
              && CodyAutoCompleteManager.isEditorInstanceSupported(editor)
              && CodyAgent.isConnected(editor.getProject())
          ? getInlineAutoCompleteItem(caret).isPresent()
          : AutoCompleteText.atCaret(caret).isPresent();
    }

    @Override
    protected void doExecute(
        @NotNull Editor editor, @Nullable Caret maybeCaret, @Nullable DataContext dataContext) {
      final Project project = editor.getProject();
      if (project == null) {
        return;
      }

      CodyAgentServer server = CodyAgent.getServer(project);
      boolean isAgentCompletion = server != null;

      if (isAgentCompletion) {
        acceptAgentAutocomplete(editor, maybeCaret);
      } else {
        Optional.ofNullable(maybeCaret)
            .or(() -> getCaret(editor))
            .flatMap(AutoCompleteText::atCaret)
            .ifPresent(
                autoComplete -> {
                  /* Log the event */
                  GraphQlLogger.logCodyEvent(project, "completion", "accepted");

                  WriteAction.run(() -> applyAutoComplete(editor.getDocument(), autoComplete));
                });
      }
    }

    private void acceptAgentAutocomplete(@NotNull Editor editor, @Nullable Caret maybeCaret) {
      Caret caret = Optional.ofNullable(maybeCaret).or(() -> getCaret(editor)).orElse(null);
      if (caret == null) {
        return;
      }
      InlineAutoCompleteItem completionItem = getInlineAutoCompleteItem(caret).orElse(null);
      if (completionItem == null) {
        return;
      }
      WriteAction.run(() -> applyInsertText(editor, caret, completionItem));
    }

    @NotNull
    private static Optional<@NotNull InlineAutoCompleteItem> getInlineAutoCompleteItem(
        Caret caret) {
      return InlayModelUtils.getAllInlaysForEditor(caret.getEditor()).stream()
          .filter(r -> r.getRenderer() instanceof CodyAutoCompleteElementRenderer)
          .map(r -> ((CodyAutoCompleteElementRenderer) r.getRenderer()).completionItem)
          .findFirst();
    }

    @NotNull
    private static Optional<Caret> getCaret(@NotNull Editor editor) {
      List<Caret> allCarets = editor.getCaretModel().getAllCarets();
      if (allCarets.size() < 2) { // Only accept completion if there's a single caret.
        return allCarets.stream().findFirst();
      } else {
        return Optional.empty();
      }
    }

    private static void applyInsertText(
        @NotNull Editor editor,
        @NotNull Caret caret,
        @NotNull InlineAutoCompleteItem completionItem) {
      Document document = editor.getDocument();
      TextRange range = EditorUtils.getTextRange(document, completionItem.range);
      document.replaceString(
          range.getStartOffset(), range.getEndOffset(), completionItem.insertText);
      caret.moveToOffset(range.getStartOffset() + completionItem.insertText.length());
    }
  }

  /**
   * Applies the autocomplete to the document at a caret. This replaces the string between the caret
   * offset and its line end with the autocompletion String and then moves the caret to the end of
   * the autocompletion.
   *
   * @param document the document to apply the autocomplete to
   * @param autoComplete the actual autocomplete text along with the corresponding caret
   */
  private static void applyAutoComplete(
      @NotNull Document document, @NotNull AutoCompleteTextAtCaret autoComplete) {
    int lineEndOffset =
        document.getLineEndOffset(document.getLineNumber(autoComplete.caret.getOffset()));
    String autoCompletionString =
        autoComplete.autoCompleteText.getAutoCompletionString(
            document.getText(TextRange.create(autoComplete.caret.getOffset(), lineEndOffset)));
    String sameLineSuffix =
        document.getText(TextRange.create(autoComplete.caret.getOffset(), lineEndOffset));
    String sameLineSuffixIfMissing =
        autoCompletionString.contains(sameLineSuffix) ? "" : sameLineSuffix;
    String finalAutoCompletionString = autoCompletionString + sameLineSuffixIfMissing;
    document.replaceString(
        autoComplete.caret.getOffset(), lineEndOffset, finalAutoCompletionString);
    autoComplete.caret.moveToOffset(
        autoComplete.caret.getOffset() + finalAutoCompletionString.length());
  }
}
