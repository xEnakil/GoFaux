# GoFaux 2.0 Development Log

This file records practical issues encountered during development and the fixes applied. It is intended for later use in the Master thesis implementation, discussion, and limitations chapters.

## 2026-05-30 - Thesis diagrams rerouted and improved

### Context

After the first generated diagram pass, the user reviewed the inserted figures and correctly pointed out that several arrows looked unprofessional, crossed awkwardly, or appeared to point to unclear targets. The problematic examples were mainly the system architecture, AI generation loop, request matching flow, and constraint workflow.

### Issue

The first generator used manual coordinate drawing. Although it produced diagrams, the layout was not good enough for a thesis document. Some diagrams tried to show too many relationships at once, which created confusing arrows. This was especially visible in the architecture diagram and in the repair/fallback parts of the AI and constraint workflows.

### Fix

The diagram generator was rewritten with a stricter grid layout and clearer orthogonal arrow routing. The revised diagrams use:

1. Simpler left-to-right process flows.
2. Grouped module boxes instead of many small arrows where a group is clearer.
3. Separate authoring and runtime lanes in the architecture diagram.
4. Clear pass/fail routing in the AI generation and validation loop.
5. Clear rule-input, prompt, validation, repair, and valid-response paths in the constraint workflow.
6. A corrected no-match path in the request matching diagram so unmatched requests flow to traffic recording instead of appearing to point into the wrong box.

The improved PNG files were regenerated in `docs/thesis/figures` and reinserted into the DOCX from a clean pre-diagram backup. Screenshot placeholders were still left untouched.

### Thesis-Relevant Lesson

Visual artifacts in a thesis need to be checked as carefully as prose. A diagram can contain the correct concepts but still weaken the thesis if the visual routing is confusing. For this reason, diagrams should prefer simple flow structure, grouped modules, and explicit pass/fail paths over dense arrows.

Thesis-ready sentence:

> The workflow diagrams were revised to use clearer process lanes and orthogonal arrows, improving readability of the architecture, request matching, AI validation, OpenAPI import, constraint handling, and project scanning explanations.

### Verification

Verification results after the improved diagram pass:

1. Seven improved PNG diagrams were generated.
2. Seven diagram placeholders were replaced in the thesis DOCX.
3. Four screenshot placeholders remain intentionally for manual screenshots.
4. Word inline shapes count: 9.
5. Word page count: 86.
6. Word tables: 13.
7. Table of contents: 1.
8. No diagram placeholders remain for Figures 1-7.
9. Microsoft Word exported the DOCX to PDF successfully.
10. The Documents plugin PNG render workflow still could not complete because LibreOffice/`soffice` is not installed in the environment.

## 2026-05-30 - Thesis workflow diagrams generated and inserted

### Context

The user asked to replace diagram placeholders in the thesis with actual generated diagrams, while leaving screenshot placeholders untouched for manual replacement later.

### Issue

The thesis contained mixed figure placeholders. Some were true diagrams, such as the design-science research process, architecture overview, request matching flow, AI generation loop, OpenAPI import workflow, constraint workflow, and project scanner workflow. Other placeholders were screenshots of the real GoFaux user interface, such as dashboard, model evaluation, traffic analytics, and executable/runtime file screenshots.

An initial DOCX edit using `python-docx` was not kept because this route can disturb Word-specific fields and citation structures. The document was restored from backup and the diagrams were inserted through Microsoft Word automation instead.

### Fix

Seven diagrams were generated as PNG files under `docs/thesis/figures` and inserted into `GoFaux_Master_Thesis_Skeleton.docx`:

1. `fig01_design_science_process.png`
2. `fig02_system_architecture.png`
3. `fig03_request_matching_flow.png`
4. `fig04_ai_generation_validation_loop.png`
5. `fig05_openapi_import_workflow.png`
6. `fig06_constraint_correction_workflow.png`
7. `fig07_project_scan_import_workflow.png`

Only the placeholders for Figures 1-7 were replaced. Screenshot placeholders were intentionally left in place.

The helper script `docs/thesis/insert_thesis_diagrams.py` now only regenerates the PNG diagram assets. The actual DOCX insertion is done through Microsoft Word automation to preserve Word fields, table of contents behavior, and the current linked numeric reference structure.

### Thesis-Relevant Lesson

The diagrams make the thesis easier to read because architecture and process explanations are no longer only textual. They also support the methodology, design, implementation, and generation chapters with visual evidence of the artifact design.

Thesis-ready sentence:

> The final thesis document includes generated workflow and architecture diagrams for the research method, GoFaux system structure, request matching, AI-assisted generation, OpenAPI import, constraint handling, and project scanning.

### Verification

Verification results after insertion:

1. Seven diagram placeholders were replaced.
2. Four screenshot placeholders remain intentionally for manual screenshots.
3. Word inline shapes count: 9.
4. Word page count: 86.
5. Word tables: 13.
6. Table of contents: 1.
7. No diagram placeholders remain for Figures 1-7.
8. Microsoft Word exported the final DOCX to PDF successfully.
9. The Documents plugin PNG render workflow still could not complete because LibreOffice/`soffice` is not installed in the environment.

## 2026-05-24 - Thesis reference system refactored to linked numeric citations

### Context

The user requested a complete refactor of the thesis referencing system. The previous version used Word footnotes that repeated source summaries inside the text. The user wanted a cleaner academic workflow: numeric citations at the end of statements, numbered references at the end of the thesis, alphabetical ordering of references, clickable in-text citation numbers, and clickable DOI/URL links inside the reference list.

### Issue

The old workflow had several practical problems:

1. Source explanations were repeated as footnotes throughout the thesis.
2. Citation notes were not clickable references.
3. The final reference list was plain text.
4. The thesis builder still contained `[[FN...]]` markers that required a separate manual conversion into Word footnotes.
5. The old approach was harder to maintain when adding or moving references.

### Fix

The thesis builder was refactored so `[[FN...]]` markers are no longer converted into footnotes. They are now converted directly into linked numeric citations such as `[4, 23, 47]`.

The reference section was changed into one numbered bibliography:

1. All 96 bibliography entries are sorted alphabetically by rendered reference text.
2. Each reference receives a stable number based on that alphabetical order.
3. Each reference paragraph has an internal Word bookmark.
4. Each in-text citation number is an internal hyperlink to its matching reference entry.
5. DOI and URL values at the end of reference entries are external Word hyperlinks.
6. The old Word footnote system was removed from the generated thesis.

The official declaration and filled university title page were preserved after regenerating the thesis, and the table of contents was checked again so imported front-matter labels did not appear in it.

### Thesis-Relevant Lesson

The updated system is closer to a conventional numeric citation style while still keeping the bibliography alphabetically ordered. It improves navigation because a reader can click a citation number in the text and jump to the numbered reference entry, then click the DOI or URL to open the source page.

Thesis-ready sentence:

> The thesis document was refactored from repeated explanatory footnotes to a linked numeric bibliography, improving navigability, maintainability, and consistency with a structured academic reference workflow.

### Verification

Verification results after the reference-system refactor:

1. Numbered references: 96.
2. Reference ordering: alphabetically sorted.
3. Internal citation hyperlinks: 229.
4. Reference bookmarks: 96.
5. Missing citation anchors: 0.
6. External DOI/URL links: 95.
7. Word footnotes: 0.
8. No `[[FN...]]` citation markers remained.
9. Old generated cover placeholder `Register number: [to be completed]` is no longer present.
10. Declaration remains page 1.
11. Filled university title page remains page 2.
12. Abstract starts on page 3.
13. Table of contents does not include declaration or title-page labels.
14. Word page count: 84.
15. Word count: 22751.
16. Tables of contents: 1.
17. Word tables: 13.
18. Microsoft Word exported the DOCX to PDF successfully.
19. The Documents plugin PNG render workflow still could not complete because LibreOffice/`soffice` is not installed in the environment.

## 2026-05-24 - Official declaration and university title page inserted

### Context

The user added two official Word template files to `docs/thesis`: a declaration of independent thesis writing and a thesis title-page template. The request was to make the declaration the first page of the thesis, make the title page the second page, fill the empty title-page fields, and keep the rest of the thesis content unchanged.

### Issue

The declaration file was already filled with the student's name, register number, field, specialization, thesis title, place, date, and signature placeholder. The title-page template still contained placeholders for field of study, thesis type, student name, thesis title, supervisor, and year.

The templates were legacy `.doc` files. When inserted into the generated thesis DOCX, Word imported some declaration/title paragraphs as `Heading 1` and also carried page-break formatting from the legacy template. This caused two practical problems:

1. Several front-matter labels were incorrectly included in the table of contents.
2. Extra blank pages appeared between the declaration, title page, and abstract.

### Fix

The title page was filled with:

1. Field of studies: `Computer Science`.
2. Thesis type: `MASTER'S THESIS`.
3. Student name: `Elmin Mughalov`.
4. Thesis title: `GoFaux 2.0: Offline AI-Assisted Mock API Server for Local Development with Automatic Response Generation and Validation`.
5. Supervisor: `Dr Hafedh Zghidi`.
6. Place and year: `Dąbrowa Górnicza 2026`.

The old generated cover page was removed from `GoFaux_Master_Thesis_Skeleton.docx`. The declaration was inserted as page 1, the filled title page was inserted as page 2, and the thesis abstract now starts on page 3.

To fix the Word-template import issues, front-matter paragraphs were changed from heading outline levels to body text outline levels so they no longer appear in the table of contents. The template's instructional line `* BACHELOR'S/BACHELOR OF ENGINEERING/MASTER'S` was removed after selecting `MASTER'S THESIS`, because it spilled onto an extra page and was no longer needed in the filled title page.

### Thesis-Relevant Lesson

Official university templates can introduce layout and style metadata that is invisible in plain-text extraction. For thesis production, it is not enough to insert the text; the merged document must also be checked for page breaks, imported heading styles, TOC pollution, and stale Word fields.

Thesis-ready sentence:

> During final document assembly, the official declaration and title-page templates were integrated into the thesis DOCX while preserving the main thesis body, footnotes, references, and table of contents structure.

### Verification

Verification results after the merge:

1. Declaration page: page 1.
2. Filled university title page: page 2.
3. Abstract starts on page 3.
4. Old generated cover placeholder `Register number: [to be completed]` is no longer present.
5. Table of contents no longer includes declaration or title-page labels.
6. Word page count: 87.
7. Word count: 22530.
8. Word footnotes: 49.
9. Tables of contents: 1.
10. Word tables: 13.
11. No `[[FN...]]` citation markers remained.
12. Microsoft Word exported the final DOCX to PDF successfully.
13. The Documents plugin PNG render workflow still could not complete because LibreOffice/`soffice` is not installed in the environment.

## 2026-05-23 - Abstract drafted and remaining thesis gaps listed

### Context

The user asked to add the thesis abstract and to identify which parts are still missing before the document can become a final submission-ready thesis. The user also reminded that future work should mention stronger codebase reading logic for detecting external API calls.

### Issue

The thesis DOCX already contained Chapters 1-9, references, footnotes, tables, and screenshot placeholders, but the `ABSTRACT` page was still empty. A check also showed that the scientific reference group was almost alphabetically sorted, but two metadata-pending entries appeared in the wrong order because the builder sorted mainly by the author field.

The user again requested intentionally imperfect grammar and wording that would not look AI-written. That part was not implemented. The abstract was written as original, natural academic prose while keeping the claims tied to the actual GoFaux artifact.

### Fix

The thesis builder and generated DOCX were updated with a four-paragraph abstract covering:

1. The local development problem caused by unavailable, paid, unstable, or hard-to-reproduce external REST APIs.
2. The GoFaux 2.0 artifact: local mock server, AI-assisted JSON generation, deterministic fallback, OpenAPI/JSON Schema support, constraint engine, project scanner, local model catalog, managed runner, and traffic analytics.
3. The design-science research approach and evaluation logic.
4. The main practical finding that local AI generation needs validation and fallback, plus future work on deeper scanner logic for external API detection.

The reference sorting logic was also changed to sort each reference group by the rendered reference text. This keeps anonymous or metadata-pending entries in a predictable alphabetical order.

### Thesis-Relevant Lesson

The abstract now frames GoFaux as a complete local-first software artifact rather than only a mock response generator. It also introduces the same cautious argument used throughout the thesis: local LLMs are helpful for authoring realistic JSON, but generated output should be validated before it becomes executable mock data.

Thesis-ready sentence:

> The abstract positions GoFaux 2.0 as a local-first mock API server that studies how AI-assisted response generation can be made practical through validation, constraints, deterministic fallback, project scanning, and request analytics.

### Remaining Thesis Gaps

The document is now structurally complete, but these parts still need final evidence or university-specific polishing:

1. Complete title-page metadata: register number, field of study, place, and final submission date.
2. Replace screenshot placeholders with real screenshots from the GoFaux dashboard, Generate page, AI Settings, model download workflow, OpenAPI import, Project Scanner, evaluation dashboard, and traffic analytics.
3. Add final measured evaluation data for manual versus AI-assisted mock creation, model comparison, JSON validity, schema adherence, constraint satisfaction, fallback usage, scanner discovery quality, and runtime performance.
4. Finalize appendix materials: prompt set, benchmark task table, screenshot set, example OpenAPI files, sample scan results, configuration examples, and development issue log.
5. Verify metadata for references marked as not yet verified in `docs/thesis/REFERENCE_MAP.md`, especially future-dated, metadata-pending, or arXiv-only entries.
6. Add a list of abbreviations or a second-language abstract if the university template requires it.
7. Run a final proofread for formatting, page numbering, figure captions, table captions, and reference consistency after screenshots and measured data are inserted.

### Verification

The thesis DOCX was regenerated, citation markers were converted into real Word footnotes, styles were reapplied, and Microsoft Word exported the document to PDF for structural checking. Verification results:

1. Word page count: 86.
2. Word count: 22361.
3. Word footnotes: 49.
4. Tables of contents: 1.
5. Word tables: 13.
6. No `[[FN...]]` citation markers remained in the DOCX.
7. The abstract and keywords are present.
8. Reference groups are alphabetically sorted: 74 scientific references, 5 technical/tool references, and 17 additional contextual references.
9. The Documents plugin PNG render workflow could not complete because LibreOffice/`soffice` is not installed in the environment. Microsoft Word PDF export was used for structural verification instead.

## 2026-05-23 - Thesis Chapters 6, 7, 8, and 9 drafted

### Context

The user asked to write the remaining thesis chapters: implementation and testing, evaluation results, discussion, and conclusion/future work. The user also asked to include screenshot placeholders where useful and to mention future improvement of project codebase reading logic for detecting external API calls more accurately.

### Issue

The thesis already described the problem, related work, methodology, architecture, and generation workflow. However, the final chapters still needed to connect the written thesis to the implemented GoFaux project. The evaluation chapter also needed to avoid inventing benchmark numbers that had not yet been measured.

The user again requested intentionally imperfect grammar and wording that would not look AI-written. That part was not implemented. The text was written in a natural but academic style, grounded in the actual project and verified facts.

### Fix

The thesis builder and DOCX were updated with full prose for:

1. `6 IMPLEMENTATION AND TESTING`
2. `7 EVALUATION RESULTS`
3. `8 DISCUSSION`
4. `9 CONCLUSION AND FUTURE WORK`

The new content includes:

1. Go project structure and implementation modules.
2. Mock engine implementation.
3. HTTP server and management endpoints.
4. Web dashboard implementation.
5. Model downloads and managed runner behavior.
6. Project scanner implementation.
7. Testing strategy and package test coverage.
8. Windows executable packaging.
9. Development challenges from the project log.
10. Evaluation environment, tasks, model comparison logic, JSON validity, constraint satisfaction, scanner evaluation, runtime performance, and traffic analytics.
11. Discussion of research questions, practical implications, comparison with WireMock/MockServer/Microcks, limitations, and future research implications.
12. Conclusion, contributions, and future work.

Future work now explicitly includes improving codebase reading logic so GoFaux can detect external API calls through custom HTTP clients, generated SDKs, service classes, environment-based base URLs, dependency injection, configuration binding, and wrapper methods that hide direct axios, fetch, requests, or net/http calls.

Additional table and screenshot placeholders were added for:

1. Implementation module overview.
2. Dashboard screenshots.
3. Local models/providers.
4. Project scanner detection categories.
5. Automated test coverage.
6. Windows executable/runtime files.
7. Prompt variants.
8. Manual versus assisted creation.
9. Local model evaluation dashboard.
10. JSON validity and schema adherence.
11. Runtime benchmark results.
12. Traffic analytics dashboard.
13. Existing tool comparison.

### Thesis-Relevant Lesson

The thesis now reaches a full draft state of roughly the intended Master thesis size. The content is still intentionally honest about evaluation limitations: the document includes verified tests, build results, implementation evidence, and placeholders for final measured benchmark tables rather than fabricating final numerical model results.

Thesis-ready sentence:

> The final chapters show that GoFaux is not only a generated mock endpoint demo, but a complete local software artifact with implementation modules, tests, packaging, evaluation hooks, traffic analytics, scanner-assisted setup, and a clear path for future research.

### Verification

The thesis DOCX was regenerated, citation markers were converted into real Word footnotes, styles were reapplied, and Microsoft Word exported the document to PDF for structural checking. Verification results:

1. Word footnotes: 49.
2. Tables of contents: 1.
3. Word page count: 85.
4. Word count: 21939.
5. Word tables: 13.
6. Chapters 6, 7, 8, and 9 and their subsections were present in the exported PDF.
7. No `[[FN...]]` markers remained in the DOCX or PDF.
8. Reference groups remained alphabetically sorted: 74 scientific references, 5 technical/tool references, and 17 additional contextual references.
9. Full Go tests passed: `go test ./...`.
10. Windows executable build passed: `go build -o dist\GoFaux.exe .`.
11. The resulting `dist\GoFaux.exe` file size was 11,892,736 bytes.

The Documents skill PNG render step was attempted but could not run because LibreOffice/soffice was not available in the local environment. Microsoft Word PDF export was used for structural checking instead.

### Files Changed

1. `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`
2. `docs/thesis/GoFaux_Master_Thesis_Skeleton.word-export.pdf`
3. `docs/thesis/build_thesis_skeleton_docx.py`
4. `docs/DEVELOPMENT_LOG.md`
5. `dist/GoFaux.exe`

## 2026-05-23 - Thesis Chapters 3, 4, and 5 drafted

### Context

The user asked to write the next main thesis parts: Chapter 3 on research methodology and requirements, Chapter 4 on system architecture and design, and Chapter 5 on AI-assisted mock generation and validation. The request also allowed screenshot or diagram placeholders where they would help the final thesis.

### Issue

The previous thesis document contained only structured headings for Chapters 3, 4, and 5. After adding the project scanner, those chapters needed to explain the research method, requirements, architecture, and generation pipeline in a way that matched the actual software. The chapter content also needed stronger methodology references, not only API and LLM references.

The user again requested intentionally imperfect grammar and wording that would not look AI-written. That part was not implemented. The text was instead written in a natural academic style with project-specific details, proper citation support, and no attempt to game AI or plagiarism detection.

### Fix

The thesis builder and DOCX were updated with full prose for:

1. `3 RESEARCH METHODOLOGY AND REQUIREMENTS`
2. `4 SYSTEM ARCHITECTURE AND DESIGN`
3. `5 AI-ASSISTED MOCK GENERATION AND VALIDATION`

The new content includes:

1. A design-science research approach.
2. Research questions and evaluation logic.
3. Functional and non-functional requirements.
4. Evaluation design and threats to validity.
5. Architecture overview, mock model, request matching, persistence, UI workflow, provider architecture, model catalog, managed runner, and project scanner architecture.
6. Generation request model, prompting strategy, local model execution, deterministic fallback, JSON validation and repair, constraint engine, OpenAPI import, and project-scan-guided generation.
7. Table placeholders for research questions, functional requirements, non-functional requirements, and mock definition fields.
8. Figure/screenshot placeholders for design-science process, system architecture, request matching, dashboard overview, generation loop, OpenAPI import, constraint workflow, and project scan workflow.

Three methodology references were added and mapped:

1. `hevner2004designScience`
2. `peffers2007dsrm`
3. `wohlin2012experimentation`

### Thesis-Relevant Lesson

The thesis now has a stronger methodological base. GoFaux is framed as a design-science artifact: a practical software engineering problem is identified, a local-first artifact is built, and the artifact is evaluated through measurable criteria such as setup effort, scanner detection, JSON validity, schema adherence, constraint satisfaction, fallback usage, latency, and runtime behavior.

Thesis-ready sentence:

> The methodology treats GoFaux as a design-science artifact whose value is evaluated through concrete local-development tasks rather than through a general claim that AI support is beneficial.

### Verification

The thesis DOCX was regenerated, citation markers were converted into real Word footnotes, styles were reapplied, and Microsoft Word exported the document to PDF for structural checking. Verification results:

1. Word footnotes: 40.
2. Tables of contents: 1.
3. Word page count: 69.
4. Word count: 15463.
5. Word tables: 4.
6. Chapters 3, 4, and 5 and their subsections were present in the exported PDF.
7. No `[[FN...]]` markers remained in the DOCX or PDF.
8. Reference groups remained alphabetically sorted: 74 scientific references, 5 technical/tool references, and 17 additional contextual references.
9. Focused project tests passed: `go test ./internal/projectscan ./internal/httpserver ./internal/mock ./internal/openapi ./internal/assistant`.

The Documents skill PNG render step was attempted but could not run because LibreOffice/soffice was not available in the local environment. Microsoft Word PDF export was used for structural checking instead.

### Files Changed

1. `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`
2. `docs/thesis/GoFaux_Master_Thesis_Skeleton.word-export.pdf`
3. `docs/thesis/build_thesis_skeleton_docx.py`
4. `docs/thesis/references.bib`
5. `docs/thesis/REFERENCE_MAP.md`
6. `docs/DEVELOPMENT_LOG.md`

## 2026-05-23 - Project folder integration scanner added

### Context

The user asked whether GoFaux could accept a project folder, analyze the source code, discover external API integrations, and automatically map those integrations into mock endpoints. A key requirement was user control: the scanner may also detect the application's own internal APIs, so the dashboard needed checkboxes to decide which detected items should actually be mocked.

### Issue

Mock creation still required either manual input, natural-language generation, or an OpenAPI document. This left a practical gap for real projects where external integrations are already visible in code through Feign clients, `RestTemplate`, `WebClient`, `axios`, `fetch`, Go `net/http`, or Python `requests`. Fully automatic import without review would be risky because internal routes and external dependencies can appear similar in source code.

### Fix

A new static project scanner was added under `internal/projectscan`. It recursively scans a selected folder while skipping heavy directories such as `.git`, `node_modules`, `vendor`, `target`, `build`, and `dist`. The scanner detects outbound client integrations and incoming/internal route declarations, then classifies each candidate with:

1. method and endpoint,
2. base URL when available,
3. source file and line,
4. client/server direction,
5. external/internal classification,
6. confidence score,
7. optional request/response DTO hints,
8. evidence text from the source line.

The HTTP server now exposes:

1. `POST /_gofaux/api/project-scan/preview`
2. `POST /_gofaux/api/project-scan/import`

The web dashboard now contains a Project Scan screen. External integrations are checked by default, internal routes are visible but unchecked, and the user can select external, select all, clear selection, or import only the checked rows. Imported rows become normal GoFaux mocks and can be generated through either the current AI provider or the deterministic local fallback.

### Verification

Unit tests were added for Java/Spring, JavaScript/TypeScript, Go, and Python detection. The focused scanner and HTTP server tests passed:

```text
go test ./internal/projectscan ./internal/httpserver
```

### Thesis-Relevant Lesson

This feature strengthens the thesis contribution by moving GoFaux from manual mock authoring toward codebase-assisted mock discovery. It also creates a useful discussion point: automatic static analysis can reduce setup effort, but human selection remains important because internal APIs and external integrations can be ambiguous without runtime context.

Thesis-ready sentence:

> GoFaux was extended with a local project scanner that discovers API integration points from source code, separates likely external dependencies from internal routes, and lets the developer approve selected candidates before generating mock responses.

### Files Changed

1. `internal/projectscan/scanner.go`
2. `internal/projectscan/scanner_test.go`
3. `internal/httpserver/projectscan.go`
4. `internal/httpserver/server.go`
5. `internal/httpserver/ui/index.html`
6. `README.md`
7. `docs/DEVELOPMENT_LOG.md`

## 2026-05-23 - Thesis updated for project scanner scope

### Context

After the project integration scanner was implemented, the thesis skeleton still described GoFaux mainly as a local AI-assisted mock authoring and validation tool. The scanner changes the thesis contribution because GoFaux can now discover likely mock targets from an existing project folder before response generation begins.

### Issue

Without updating the thesis, the written scope would understate the implemented artifact. The old structure covered OpenAPI import, JSON Schema constraints, local models, analytics, and model evaluation, but it did not explicitly cover source-code-based discovery, scanner confidence, external/internal classification, or the human approval step before importing detected integrations as mocks.

### Fix

The thesis skeleton was updated so the scanner appears as a first-class part of the project:

1. Chapter 1 now includes codebase-assisted mock discovery in the research problem, aim, scope, contribution, research questions, and hypotheses.
2. Chapter 2 now includes `2.6 Source-code scanning and API discovery`.
3. The former LLM, structured-output, local-model, and literature-gap subsections were renumbered to `2.7` through `2.10`.
4. Chapters 4, 5, 6, and 7 now include dedicated scanner sections:
   - `4.8 Project scanner architecture`
   - `5.8 Project-scan-guided mock creation`
   - `6.6 Project scanner implementation`
   - `7.7 Project scanner discovery evaluation`
5. The list of tables and list of figures now include scanner-related items.
6. Two scanner-related references were added: Respector for static REST API specification generation and SafeRESTScript for static checking of REST API consumers.

### Thesis-Relevant Lesson

The scanner strengthens GoFaux as a research artifact because it adds a discovery stage before generation. The thesis can now evaluate not only whether local AI can produce valid mock JSON, but also whether local static analysis can reduce setup effort by identifying candidate API dependencies from source code.

Thesis-ready sentence:

> The addition of project scanning extends GoFaux from an AI-assisted mock response generator into a codebase-assisted mock setup tool, where source-code evidence, user approval, local generation, validation, and serving are combined in one workflow.

### Verification

The thesis DOCX was regenerated, citation markers were converted into real Word footnotes, the table of contents was updated, and Microsoft Word exported the document to PDF for structural checking. Verification results:

1. Focused scanner tests passed: `go test ./internal/projectscan ./internal/httpserver`.
2. Word footnotes: 22.
3. Tables of contents: 1.
4. Word page count: 53.
5. Word count: 8857.
6. Chapter 2 scanner subsection and scanner sections in Chapters 4-7 were present in the exported PDF.
7. No `[[FN...]]` markers remained in the DOCX or PDF.
8. Reference groups remained alphabetically sorted: 71 scientific references, 5 technical/tool references, and 17 additional contextual references.

The Documents skill PNG render step was attempted but could not run because LibreOffice/soffice was not available in the local environment. Microsoft Word PDF export was used for structural checking instead.

### Files Changed

1. `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`
2. `docs/thesis/GoFaux_Master_Thesis_Skeleton.word-export.pdf`
3. `docs/thesis/build_thesis_skeleton_docx.py`
4. `docs/thesis/references.bib`
5. `docs/thesis/REFERENCE_MAP.md`
6. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - Local model selected but no mock was created

### Context

The application could download a GGUF model file through the local model catalog. TinyLlama 1.1B Chat Q4_K_M was downloaded successfully into `.gofaux/models`.

After selecting the model, the user expected `/people` or similar endpoints to exist immediately. However, `View mocks` showed:

```text
No mocks added yet.
```

### Issue

Selecting or downloading an AI model only configures the generation backend. It does not automatically create a mock endpoint. The previous generation attempt also failed because the user typed `list` as the HTTP method, which produced an unsupported method:

```text
unsupported method "LIST"
```

As a result, no mock was saved and the server had no endpoint to serve.

### Fix

The CLI was improved in several ways:

1. `View mocks` now explains that option `2` should be used to generate a mock when the list is empty.
2. Starting the server with zero mocks now warns that requests will return `404`.
3. The generated-mock flow now asks for natural language first, for example:

```text
list people with name and age
```

4. The app infers a valid HTTP method and endpoint from that phrase:

```text
GET /people
```

5. After choosing the GoFaux managed runner, the app offers to generate a mock immediately.
6. Switching to OpenAI-compatible mode no longer reuses a `.gguf` file path as the model name.

### Thesis-Relevant Lesson

This issue shows the difference between **model configuration** and **mock creation**. A local AI model can assist mock generation, but the application still needs a clear workflow that guides the user from model selection to endpoint creation.

Thesis-ready sentence:

> During interactive testing, a usability issue was identified: users could interpret model selection as mock creation. The workflow was therefore adjusted to distinguish AI backend configuration from endpoint generation and to infer valid HTTP methods from natural-language intents.

### Files Changed

1. `internal/cli/menu.go`
2. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - Managed runner executable existed but failed to start

### Context

The user downloaded TinyLlama successfully and selected the GoFaux managed runner. The runner files were present in `.gofaux/runners/llama.cpp`, including `llama-server.exe`.

During mock generation, GoFaux attempted to start the local runner and failed with:

```text
fork/exec .gofaux\runners\llama.cpp\llama-server.exe: The system cannot find the path specified.
```

### Issue

The runner launcher passed a relative executable path to `exec.CommandContext`. On Windows, process startup can fail when relative paths, working directories, spaces, and special characters in the full workspace path interact. The executable existed, but the startup path resolution was not robust.

There was also a small usability issue in the generation prompt: the user typed `list` into the endpoint field, which would have replaced the inferred `/people` endpoint with `/list`.

### Fix

The runner startup now resolves both paths to absolute paths before launching:

1. The GGUF model path is converted to an absolute path.
2. The `llama-server.exe` path is converted to an absolute path.
3. The launcher validates that both files exist before starting the process.

The CLI endpoint prompt now protects the inferred endpoint. If the user types an action word such as `list`, GoFaux keeps the inferred endpoint, for example `/people`.

The fields prompt was also made more forgiving. If the user writes a natural phrase mentioning age instead of `name:type` syntax, GoFaux maps it to a useful people DTO hint.

### Thesis-Relevant Lesson

This issue is a practical example of platform-specific behavior in local AI tooling. Managing local inference is not only a model-selection problem; it also requires robust process management, path handling, and user guidance.

Thesis-ready sentence:

> During Windows testing, the managed-runner feature exposed a path-resolution problem: the runner executable existed but could not be started reliably through a relative path. The implementation was corrected to use absolute paths for both the model file and runner executable, improving portability of local inference startup.

### Files Changed

1. `internal/runner/manager.go`
2. `internal/cli/menu.go`
3. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - Local model echoed the instruction DTO

### Context

The user generated a mock for:

```text
GET /people
```

The expected response was a list of realistic person objects with fields such as `name`, `age`, `gender`, and `hairColor`. Instead, the endpoint returned the internal generation specification:

```json
{
  "task": "generate_mock_response_json",
  "endpoint": "/people",
  "method": "GET",
  "preferred_shape": "array_or_object_with_items_array"
}
```

### Issue

Tiny local instruction-tuned models can sometimes echo the prompt or specification instead of completing the task. The previous validation only checked whether the model output was syntactically valid JSON. Because the echoed specification was valid JSON, GoFaux saved it as the mock response body.

This showed that JSON parsing alone is not enough for AI-assisted mock generation. The application also needs semantic quality checks that detect whether the response is useful for the intended endpoint.

### Fix

The generation flow now detects instruction echoes before saving a mock. If the model returns keys such as `task`, `rules`, or `preferred_shape`, GoFaux treats the result as a failed generation and uses its local structured fallback.

The prompt was also strengthened to tell the model that the specification is not the output. For people-list requests, GoFaux now asks for 10 to 20 realistic person records and provides an expected response shape.

The already-saved `/people` mock was repaired so it returns a real payload:

```json
{
  "items": [
    {
      "id": 1,
      "name": "Ava Johnson",
      "age": 21,
      "gender": "female",
      "hairColor": "brown",
      "email": "ava.johnson@example.com"
    }
  ],
  "total": 12
}
```

### Thesis-Relevant Lesson

This issue demonstrates a central engineering challenge of local AI-assisted development tools: small offline models may produce valid but semantically incorrect output. Robust mock generation therefore requires layered validation: syntactic validation, semantic checks, and deterministic local fallback behavior.

Thesis-ready sentence:

> During testing with a local GGUF model, the model returned the prompt specification itself as valid JSON. This revealed that syntactic JSON validation is insufficient for AI-generated mock APIs; the implementation was extended with semantic echo detection and deterministic fallback generation to preserve endpoint usefulness.

### Files Changed

1. `internal/assistant/prompt.go`
2. `internal/assistant/quality.go`
3. `internal/assistant/quality_test.go`
4. `internal/assistant/template.go`
5. `internal/cli/menu.go`
6. `gofaux.mocks.json`
7. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - Generalizing AI mock generation beyond hardcoded people lists

### Context

After fixing the prompt-echo issue, the user clarified that the goal is not to hardcode a `/people` endpoint or always return a fixed number of records. The intended behavior is that a local model receives a simple natural-language request and decides the right JSON shape:

```text
list products with price and status
```

should produce a collection, while:

```text
user with name and age
```

should produce a single object when the endpoint is singular or parameterized.

### Issue

The first fallback implementation was too specific. It handled the observed `/people` failure correctly, but it risked making the system look like a collection of hardcoded generators rather than an AI-assisted mock API tool.

The prompt also mentioned a specific people-list shape and count range, which could bias the model toward one example instead of letting it infer the response from the user's instruction.

### Fix

The generation prompt was rewritten to be intent-based. It now asks the local model to infer:

1. Whether the response should be a list or one object.
2. Which fields belong in the JSON from the user intent, endpoint, DTO sample, schema, or structured field hints.
3. How many list items are appropriate, using an explicit count only when the user gives one.

The semantic validation flow now gives the model one repair attempt if it echoes the request details. The deterministic fallback is used only after the model fails to correct itself.

The fallback generator was also generalized. It can infer reasonable local examples for resources such as people, users, products, orders, books, cars, and generic entities. It respects explicit counts such as `three products`, keeps endpoints like `/users/{id}` as single-object responses, and appends natural field hints to the prompt when they are not written in strict `name:type` form.

### Thesis-Relevant Lesson

This iteration separates **AI-first generation** from **deterministic resilience**. The local model remains responsible for creating the mock response, while the application provides guardrails, repair prompts, and fallback behavior to keep the workflow usable on small offline models.

Thesis-ready sentence:

> The mock-generation workflow was refined from endpoint-specific fallback logic into an intent-based pipeline. The local model is prompted to infer list versus object shape and field structure from natural language, while deterministic generation is retained only as a resilience mechanism after validation or repair failure.

### Files Changed

1. `internal/assistant/intent.go`
2. `internal/assistant/prompt.go`
3. `internal/assistant/template.go`
4. `internal/assistant/quality.go`
5. `internal/assistant/template_test.go`
6. `internal/assistant/quality_test.go`
7. `internal/cli/menu.go`
8. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - Valid JSON was still semantically wrong

### Context

The local managed TinyLlama model generated a response for this instruction:

```text
list of people, like maybe 25 amount, and their nationality and percentage random from 50 to 100 and passport number all string starting with WS
```

The model returned valid JSON, but the body was only partially correct:

1. It returned 6 records instead of 25.
2. It used a `dtos` wrapper instead of the application list contract.
3. It included request metadata such as `status_code` and `user_intent` in the body.
4. Percentage values were outside the requested 50 to 100 range.
5. Passport values did not start with `WS`.

### Issue

This was not a syntax error or a prompt echo. It was a semantic quality problem: the model understood the general idea but failed several concrete constraints. This is especially likely with very small local models, which can produce plausible but inconsistent JSON.

The endpoint inference also exposed a small natural-language parsing issue. The phrase `list of people` was initially inferred as `/of` because the parser selected the word immediately after `list`.

### Fix

The generation pipeline now performs semantic quality checks after JSON validation. It checks for:

1. Leaked request metadata fields such as `status_code`, `user_intent`, `endpoint`, and `method`.
2. Proper list response shape using an `items` array and `total` count.
3. Exact item count when the user provides one, for example 25 records.
4. Percentage constraints such as values between 50 and 100.
5. String prefix constraints such as passport numbers starting with `WS`.

If the model fails these checks, GoFaux asks the model to repair the response with the specific validation failures. Only if the model still fails does GoFaux use the deterministic local fallback.

The fallback generator was also improved so it can satisfy common constraints such as nationality, percentage ranges, and passport prefixes. Endpoint inference now skips filler words like `of`, `the`, `all`, and `maybe`, so `list of people` correctly infers `/people`.

### Thesis-Relevant Lesson

This issue provides a strong example of why AI-assisted developer tools need semantic validation. For mock APIs, the correctness criterion is not simply valid JSON; the response must also satisfy the user's domain constraints.

Thesis-ready sentence:

> A later test showed that valid JSON could still be semantically incorrect: the local model generated a plausible list response but violated count, range, prefix, and metadata-exclusion constraints. The implementation was therefore extended with semantic quality checks and targeted repair prompts before falling back to deterministic generation.

### Files Changed

1. `internal/assistant/intent.go`
2. `internal/assistant/prompt.go`
3. `internal/assistant/quality.go`
4. `internal/assistant/quality_test.go`
5. `internal/assistant/template.go`
6. `internal/assistant/template_test.go`
7. `internal/cli/menu.go`
8. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - Moving from terminal tool to local UI application

### Context

The user asked whether Python would be easier than Go for the project, and whether GoFaux could become a more complete tool with a polished interface and a double-clickable Windows executable.

### Decision

Python would be convenient for rapid AI experiments because the machine-learning ecosystem is larger and easier to prototype with. However, Go remains a strong choice for this thesis project because it can package the mock server, local model management, admin API, embedded UI, and process supervision into a single native executable.

The implementation therefore keeps Go as the main application runtime and exposes AI generation through local provider abstractions. This keeps the project reproducible for users who do not want to install Python environments.

### Fix

GoFaux was extended with a browser-based local dashboard served from the same executable:

1. The app now starts in UI mode by default.
2. The old terminal menu remains available with `--cli`.
3. The UI is embedded into the Go binary and served at `/_gofaux/ui/`.
4. Admin APIs were added under `/_gofaux/api/...`.
5. Users can view mocks, inspect response bodies, test endpoints, delete mocks, create manual mocks, generate mocks with local AI, and update local AI settings from the dashboard.
6. The AI generation workflow was moved into a shared `internal/generator` module so UI and CLI behavior can converge over time.
7. The project can now be built into a Windows executable at `dist/GoFaux.exe`.

### Thesis-Relevant Lesson

This iteration reframes GoFaux from a command-line prototype into a local developer tool. The browser UI improves usability without sacrificing the local-first architecture because the interface, mock server, AI provider configuration, and model runner orchestration are still hosted by the same local executable.

Thesis-ready sentence:

> Although Python offers a richer AI experimentation ecosystem, Go was retained as the primary implementation language because it enables a portable local developer tool: the mock server, AI-generation workflow, embedded dashboard, configuration storage, and runner supervision can be distributed as a single Windows executable.

### Files Changed

1. `internal/app/app.go`
2. `internal/httpserver/server.go`
3. `internal/httpserver/ui/index.html`
4. `internal/generator/generator.go`
5. `internal/generator/intent.go`
6. `internal/generator/generator_test.go`
7. `internal/mock/store.go`
8. `README.md`
9. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - Managed runner connection closed during UI generation

### Context

During generation from the browser UI, the managed local runner returned a Windows socket error:

```text
read tcp 127.0.0.1:63984->127.0.0.1:39281: wsarecv: An existing connection was forcibly closed by the remote host
```

GoFaux fell back to deterministic generation, but the UI displayed the provider failure in a way that felt like an application crash.

### Issue

The app was technically resilient because it produced a fallback response, but the experience was not polished. The message exposed low-level network details and did not clearly communicate that GoFaux had recovered.

The managed llama.cpp-compatible runner was also being called with OpenAI's `response_format` option. Some local model servers or builds can behave poorly with unsupported or partially supported request options, especially with small GGUF models and CPU-only runners.

### Fix

Managed-runner generation now avoids the `response_format` parameter and relies on prompt-level JSON instructions plus GoFaux's own JSON extraction and semantic validation.

The generator also retries managed-runner failures once by restarting the runner. If the retry still fails, GoFaux continues with deterministic local generation and returns clear UI messages explaining what happened.

The runner now writes stdout/stderr to:

```text
.gofaux/runners/llama.cpp/llama-server.log
```

This makes future local inference failures easier to diagnose. The UI message styling was also improved so recovery messages are shown as controlled status messages instead of looking like an unhandled crash.

### Thesis-Relevant Lesson

This issue demonstrates a practical reliability challenge in local AI tools: model inference may fail for reasons outside the application logic, such as native runtime instability, unsupported API parameters, or process termination. A robust local tool should degrade gracefully and preserve the user's workflow.

Thesis-ready sentence:

> When the managed local inference server closed a connection during generation, the application recovered through fallback generation but initially presented the event as a raw provider failure. The workflow was improved with managed-runner retry, runner logs, and clearer UI recovery messages, illustrating the need for graceful degradation in local AI-assisted developer tools.

### Files Changed

1. `internal/assistant/openai_compatible.go`
2. `internal/generator/generator.go`
3. `internal/runner/manager.go`
4. `internal/cli/menu.go`
5. `internal/httpserver/ui/index.html`
6. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - UI needed model management and request analytics

### Context

The user wanted the browser dashboard to cover the same practical workflow as the terminal application and to feel like a professional local developer tool. Two missing areas were especially important:

1. Downloading and selecting local AI models directly from the UI.
2. Seeing what external applications send to GoFaux when calling mocked endpoints, including request bodies.

### Issue

The first dashboard exposed basic mock creation and settings, but model management still depended on terminal workflows. The dashboard also only showed request counts, not actual traffic details. This limited its usefulness for debugging backend integrations, where seeing the incoming request method, path, headers, query string, body, matched mock, and response status is central.

### Fix

The dashboard was restructured into a fuller local console with separate areas for Overview, Generate, Mocks, Traffic, Models, and Settings.

Backend additions:

1. Added an in-memory traffic recorder for mock requests.
2. Captured request method, path, query, headers, body, status, duration, response size, matched mock, and path parameters.
3. Added analytics endpoints for request summaries and clearing runtime traffic.
4. Added model download jobs with progress tracking.
5. Added UI endpoints for catalog model downloads and custom GGUF URL downloads.
6. Extended downloaded-model listing to include custom `.gguf` files in the local model directory.

UI additions:

1. A more professional multi-section layout.
2. Request timeline and distribution charts.
3. Recent traffic and detailed request inspection.
4. Model catalog cards with download/use actions.
5. Download progress bars.
6. Custom GGUF model download form.
7. More complete manual mock creation fields, including headers, required query parameters, required request headers, delay, and priority.

The browser's `/favicon.ico` request is now ignored so it does not pollute the traffic log.

### Thesis-Relevant Lesson

This iteration moved the application from a mock authoring interface toward a local observability tool. For backend mock servers, analytics are not optional decoration: developers need to inspect what their client application actually sends, especially request bodies and headers, to design realistic mocks and debug integration behavior.

Thesis-ready sentence:

> The dashboard was extended with local model management and runtime traffic analytics, allowing users to download GGUF models, select them for managed inference, and inspect incoming mock requests including headers, query strings, bodies, matched mock definitions, and response status. This strengthened GoFaux's role as both a mock authoring tool and a local backend integration observability tool.

### Files Changed

1. `internal/httpserver/analytics.go`
2. `internal/httpserver/downloads.go`
3. `internal/httpserver/server.go`
4. `internal/httpserver/ui/index.html`
5. `internal/modelhub/manager.go`
6. `README.md`
7. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - Managed local runner closed connections during UI generation

### Context

While generating a mock from the UI with the managed TinyLlama GGUF model, GoFaux showed:

```text
Managed runner request failed: Post "http://127.0.0.1:39281/v1/chat/completions": read tcp ... wsarecv: An existing connection was forcibly closed by the remote host.
Restarting the managed runner and retrying once.
Managed runner stayed unavailable after retry...
```

The user experience looked like an AI crash even though the application recovered through deterministic fallback.

### Issue

The managed `llama-server.exe` process was started with a short-lived startup context. After the server reported healthy, the caller cancelled that context, which could terminate the native runner just as the UI sent the generation request. The runner was also started with aggressive defaults for a tiny local model, including a 4096-token context for a model trained around 2048 context tokens and multi-slot server behavior.

After the socket-level issue was fixed, TinyLlama still sometimes returned prose, markdown, or a single-item list. Those are common limitations of very small local models and need application-level validation.

### Fix

The managed runner lifecycle was changed so readiness timeout no longer owns the native process lifetime. GoFaux now starts `llama-server.exe` independently and shuts it down through explicit cleanup after generation.

The runner now starts with more conservative local defaults:

1. 2048 context size.
2. Single server slot.
3. Smaller batch and micro-batch sizes.
4. Prompt cache disabled.
5. Flash attention disabled.
6. GPU/offload paths forced off for the CPU runner.

The OpenAI-compatible provider now sends a `max_tokens` limit and uses a stricter system instruction. If the model's first answer does not contain valid JSON, GoFaux retries with a compact prompt designed for small local models.

List validation was also updated to accept both common API list shapes:

1. A top-level JSON array.
2. A JSON object with `items` and `total`.

For open-ended list prompts, GoFaux now rejects one-item list responses unless the user explicitly asked for one item.

### Verification

End-to-end UI/API generation was tested through `/_gofaux/api/generate` using the managed TinyLlama model with this prompt:

```text
list people with name and age and gender and hair color
```

The result used the managed local model, did not fall back to the deterministic template, and produced a valid multi-person JSON array.

### Thesis-Relevant Lesson

This issue shows that local AI integration has two separate reliability layers:

1. Native inference process reliability: the application must manage runner lifecycle, startup, shutdown, and conservative hardware defaults.
2. Model output reliability: small local models need validation, retry prompts, and fallback paths because they may answer with prose, markdown, wrong shapes, or too few objects.

Thesis-ready sentence:

> A managed local AI runner introduces both systems-level and semantic reliability challenges: the application must keep the native inference process alive independently of startup timeouts, and it must validate model output because small local models may return prose, markdown, or incomplete list responses even when instructed to return JSON.

### Files Changed

1. `internal/runner/manager.go`
2. `internal/assistant/openai_compatible.go`
3. `internal/assistant/prompt.go`
4. `internal/assistant/quality.go`
5. `internal/assistant/quality_test.go`
6. `docs/DEVELOPMENT_LOG.md`

## 2026-05-13 - Passport pattern handling and stronger local model choices

### Context

The user generated a people-list mock with this instruction:

```text
list people with name, age and nationality, hair color, and passport number with WS-xxxxxxx pattern
```

TinyLlama copied internal request metadata during one attempt, which GoFaux correctly rejected. The deterministic fallback then returned valid JSON, but passport numbers used the default `P1000000` style instead of the requested `WS-xxxxxxx` style.

### Issue

Two problems appeared:

1. The fallback generator only understood prefix phrases like `starting with WS`; it did not understand placeholder patterns like `WS-xxxxxxx`.
2. The UI showed too many low-level validator messages when a model copied request metadata, which made a controlled recovery look more severe than it was.

TinyLlama also showed its practical quality limit: it can run locally, but it is weak at reliably following structured JSON instructions.

### Fix

The value-rule parser now recognizes passport placeholder patterns such as `WS-xxxxxxx`, extracting both the string prefix and the expected number of digits. The deterministic generator now emits matching values like `WS-1000000`.

The validator now summarizes request-metadata copying instead of listing every internal key separately, and it skips extra list-shape checks when the response is clearly an instruction echo.

The curated local model catalog was expanded with stronger GGUF choices:

1. Qwen2.5 1.5B Instruct Q4_K_M.
2. Qwen2.5 3B Instruct Q4_K_M.
3. Phi-3 Mini 4K Instruct Q4.

These models are heavier than TinyLlama but should follow JSON-generation instructions better on machines with enough memory.

### Thesis-Relevant Lesson

This iteration demonstrates why a local AI mock generator should combine model choice, deterministic fallback, and semantic validators. The AI model can provide flexible generation, while deterministic constraints preserve correctness for important domain rules such as identifier formats.

Thesis-ready sentence:

> The passport-pattern issue showed that robust AI-assisted mock generation requires deterministic constraint handling alongside local model inference: when a small model failed to follow a `WS-xxxxxxx` requirement, the application-level parser and fallback generator were extended to enforce the format independently of the model.

### Files Changed

1. `internal/assistant/intent.go`
2. `internal/assistant/prompt.go`
3. `internal/assistant/quality.go`
4. `internal/assistant/template.go`
5. `internal/assistant/intent_test.go`
6. `internal/assistant/template_test.go`
7. `internal/modelhub/catalog.go`
8. `docs/DEVELOPMENT_LOG.md`

## 2026-05-20 - Thesis-grade OpenAPI import, constraint engine, and model evaluation dashboard

### Context

The user asked for three larger enhancements that would make the project stronger as a master's thesis artifact:

1. A Local AI Model Evaluation Dashboard.
2. OpenAPI/Swagger import.
3. A Constraint Engine.

The goal was not only to add features, but to create material for a 70-page thesis: architecture decisions, validation logic, experiments, comparison tables, and reproducible workflows.

### Issue

GoFaux could generate mocks from natural-language prompts, but three research gaps remained:

1. There was no contract-driven workflow for importing existing API definitions.
2. Constraint handling was partly hidden inside ad hoc generation helpers instead of being an explicit subsystem.
3. There was no way to compare local models, deterministic fallback, latency, JSON validity, semantic quality, or fallback usage on the same benchmark prompts.

Without these features, the thesis would mostly describe an implementation. With them, it can also describe evaluation, validation, and comparison.

### Fix

#### Constraint Engine

The generation request model now supports structured constraints and manual constraint text. Examples:

```text
passportNumber: pattern WS-xxxxxxx
age: integer 18-70
email: email
price: number 10-500
status: enum pending, approved, rejected
```

The engine can now:

1. Parse manual constraint text.
2. Infer constraints from natural-language descriptions.
3. Extract constraints from JSON Schema/OpenAPI response schemas.
4. Add constraint summaries to prompts sent to local AI models.
5. Validate generated JSON against constraints.
6. Use constraints in the deterministic fallback generator.

The deterministic fallback now prioritizes semantic constraints before generic type constraints. This matters for fields such as `age`, where a schema may say both `integer` and `18-70`; the fallback must generate an integer inside the range, not merely any integer.

#### OpenAPI / Swagger Import

A new OpenAPI parser was added. It supports JSON and YAML OpenAPI/Swagger documents and can:

1. Read API metadata.
2. Extract paths, methods, operation IDs, summaries, response status codes, request schemas, and response schemas.
3. Resolve local `$ref` schema references.
4. Extract DTO fields from response schemas.
5. Convert schema rules into constraint-engine rules.
6. Preview operations in the UI.
7. Import selected operations as saved GoFaux mocks.
8. Import either with deterministic local generation or the currently selected AI provider.

The UI now has an OpenAPI workspace where the user can paste a contract, preview operations, select endpoints, and import them as mocks.

#### Local AI Model Evaluation Dashboard

A new asynchronous evaluation workflow was added. The backend can run benchmark cases across selected targets:

1. Template fallback.
2. Current configured provider.
3. Downloaded managed GGUF models.

For each target/case pair, GoFaux records:

1. JSON validity.
2. Semantic quality pass/fail.
3. Whether the model itself passed without fallback.
4. Fallback usage.
5. Attempts.
6. Latency.
7. Validation issues.
8. Generated body preview.

The UI now has an Evaluation workspace that lets the user select targets, edit benchmark cases as JSON, start evaluation jobs, poll progress, and inspect summary tables.

### Verification

Automated tests were added for:

1. Manual constraint parsing.
2. Constraint validation.
3. Constraint-aware template generation.
4. OpenAPI YAML parsing.
5. `$ref` and schema-derived field/constraint extraction.

Smoke tests were also run against the local HTTP API using the template provider:

1. `POST /_gofaux/api/openapi/preview` detected one operation and extracted schema constraints.
2. `POST /_gofaux/api/openapi/import` created a mock from an OpenAPI people-list operation.
3. The imported mock generated passport numbers like `WS-1000000`.
4. `POST /_gofaux/api/evaluations` started an evaluation job.
5. `GET /_gofaux/api/evaluations/{id}` returned a completed job with a model pass.

Full Go tests passed with:

```text
go test ./...
```

### Thesis-Relevant Lesson

This iteration turns GoFaux from a local mock generator into an experimental platform for local AI-assisted API mocking. The OpenAPI importer supports contract-driven generation, the constraint engine provides deterministic correctness checks, and the evaluation dashboard enables empirical comparison of local models.

Thesis-ready sentence:

> The project was extended with contract-driven mock generation, an explicit constraint engine, and an evaluation dashboard, enabling GoFaux to function not only as a developer tool but also as a research artifact for comparing local language models on JSON validity, schema adherence, semantic constraint satisfaction, latency, and fallback frequency.

### Files Changed

1. `go.mod`
2. `go.sum`
3. `internal/assistant/constraints.go`
4. `internal/assistant/constraints_test.go`
5. `internal/assistant/types.go`
6. `internal/assistant/prompt.go`
7. `internal/assistant/quality.go`
8. `internal/assistant/template.go`
9. `internal/generator/generator.go`
10. `internal/openapi/importer.go`
11. `internal/openapi/importer_test.go`
12. `internal/httpserver/server.go`
13. `internal/httpserver/openapi.go`
14. `internal/httpserver/evaluations.go`
15. `internal/httpserver/ui/index.html`
16. `README.md`
17. `docs/DEVELOPMENT_LOG.md`

## 2026-05-20 - Thesis reference pack added

### Context

The user asked how to make the thesis more science-based and whether references should primarily be scientific papers. The project needed a dedicated place where future thesis-writing work can find open references and map them directly to GoFaux features.

### Issue

Without a curated reference pack, a later AI writing the thesis might cite generic or weak sources, mix scientific papers with tool documentation, or fail to connect references to concrete implementation choices such as OpenAPI import, JSON Schema validation, constraint handling, local model evaluation, and API mocking.

### Fix

Two thesis support files were added:

1. `docs/thesis/references.bib`
2. `docs/thesis/REFERENCE_MAP.md`

The BibTeX file includes the user-provided papers plus open scientific references and official specifications relevant to:

1. REST API testing.
2. API mocking.
3. JSON Schema and structured validation.
4. OpenAPI-driven workflows.
5. Local/offline LLMs.
6. LLM evaluation.
7. AI-assisted developer productivity.

The reference map links project features to citation keys and thesis chapters. It also distinguishes between scientific papers, technical specifications, and tool documentation so the thesis can rely mainly on academic sources while still citing standards and tool docs where appropriate.

### Thesis-Relevant Lesson

The reference pack helps turn GoFaux from an implementation artifact into a thesis-ready research artifact. It makes the relationship between system features, academic literature, and evaluation methodology explicit.

Thesis-ready sentence:

> A dedicated reference map was created to connect each GoFaux feature to relevant scientific literature, technical standards, and evaluation methodology, supporting a more rigorous thesis structure and reducing the risk of weak or generic citations.

### Files Changed

1. `docs/thesis/references.bib`
2. `docs/thesis/REFERENCE_MAP.md`
3. `docs/DEVELOPMENT_LOG.md`

## 2026-05-23 - Chapter 2 background and related work drafted

### Context

The user asked to write Chapter 2 of the thesis, covering REST APIs, local development workflows, REST API testing, test doubles, existing mock server tools, OpenAPI, JSON Schema, LLMs for software engineering, structured output generation, local models, and the literature gap.

### Issue

The chapter needed to connect many reference areas without becoming a generic AI literature review. It also needed to keep references accurate: scientific papers were used for research claims, while OpenAPI, JSON Schema, WireMock, MockServer, and Microcks were treated as specifications or tool documentation rather than peer-reviewed evidence.

The user requested intentionally imperfect grammar and wording that would not look AI-written. That part was not implemented. Instead, the chapter was written in a natural academic style with original wording, varied sentence structure, and proper citation support.

### Fix

Chapter 2 was added to the thesis builder and regenerated into `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`. The chapter now contains full prose for:

1. `2.1 REST APIs and local development workflows`
2. `2.2 REST API testing and integration challenges`
3. `2.3 Test doubles, API mocking, and service virtualization`
4. `2.4 Existing mock server tools`
5. `2.5 OpenAPI and JSON Schema`
6. `2.6 Large language models for software engineering`
7. `2.7 Structured output generation and schema-constrained LLMs`
8. `2.8 Local and small language models`
9. `2.9 Literature gap`

Sixteen new Word footnotes were added for Chapter 2, bringing the document total to 21 footnotes. The citations cover REST API testing, mock generation, service virtualization, OpenAPI, JSON Schema, structured output generation, local/offline LLMs, small model families, and LLM-assisted API research.

### Thesis-Relevant Lesson

The related work chapter now supports the core argument that GoFaux is not only a mock server and not only an AI demo. It is positioned as an integrated local-first workflow that combines mock serving, AI-assisted response authoring, schema validation, deterministic fallback, request analytics, and model comparison.

Thesis-ready sentence:

> The reviewed literature shows that API testing, mocking, schema validation, structured LLM output, and local model execution are well-established individual areas, but fewer tools combine them into a single local workflow for generating, validating, serving, and evaluating mock REST responses.

### Verification

The thesis DOCX was regenerated, citation markers were converted into real Word footnotes, the table of contents was updated, and Microsoft Word exported the document to PDF for structural checking. Verification results:

1. Word footnotes: 21.
2. Tables of contents: 1.
3. Word page count: 49.
4. Word count: 8201.
5. Chapter 2 headings and text were present in the exported PDF.
6. No `[[FN...]]` markers remained in the DOCX.
7. Reference groups remained alphabetically sorted: 69 scientific references, 5 technical/tool references, and 17 additional contextual references.

The Documents skill PNG render step was attempted but could not run because LibreOffice/soffice was not available in the local environment. Microsoft Word PDF export was used for structural checking instead.

### Files Changed

1. `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`
2. `docs/thesis/GoFaux_Master_Thesis_Skeleton.word-export.pdf`
3. `docs/thesis/build_thesis_skeleton_docx.py`
4. `docs/DEVELOPMENT_LOG.md`

## 2026-05-23 - Master thesis structure planning DOCX created

### Context

The user asked for a Word document that follows the university thesis rules and the EEMCS example structure, but acts as a planning skeleton rather than the final written thesis. The document needed a table of contents, formal headings, planned page budgets, blank writing spaces after headings, and reference planning for the GoFaux project.

### Issue

The thesis needed a practical writing scaffold that satisfies formatting expectations before the full text is written. A plain Markdown outline would not be enough because the university rules specify page size, font, line spacing, margins, chapter title style, subchapter title style, page numbering, references, figures, tables, and appendices.

### Fix

A reproducible DOCX builder was added and used to generate:

1. `docs/thesis/GoFaux_Master_Thesis_Structure_Plan.docx`
2. `docs/thesis/build_thesis_plan_docx.py`

The generated document uses:

1. A4 page size.
2. Times New Roman 12 pt body text.
3. 1.5 line spacing.
4. 3.5 cm left margin and 2.5 cm top, bottom, and right margins.
5. Bold uppercase 14 pt chapter headings.
6. Bold 12 pt subchapter headings.
7. A planned 78-page main-text structure.
8. A static planned table of contents based on the EEMCS example.
9. Placeholder writing space after every planned section.
10. Reference-route notes using `docs/thesis/references.bib` and `docs/thesis/REFERENCE_MAP.md`.

The document was exported through Microsoft Word to PDF for structural checking. The packaged LibreOffice PNG renderer could not run because LibreOffice was not available in the environment, so full PNG visual QA was not completed.

### Thesis-Relevant Lesson

This document becomes the thesis control structure. It separates the work into page-budgeted writing units, helping future thesis-writing sessions fill one section at a time while staying inside the 70-90 page Master thesis requirement.

Thesis-ready sentence:

> A thesis structure template was prepared according to the university formatting rules, with each chapter and subchapter assigned a target page budget and connected to the relevant implementation artifacts and scientific references.

### Files Changed

1. `docs/thesis/GoFaux_Master_Thesis_Structure_Plan.docx`
2. `docs/thesis/build_thesis_plan_docx.py`
3. `docs/DEVELOPMENT_LOG.md`

## 2026-05-23 - Clean master thesis skeleton DOCX created

### Context

The first thesis document was useful as a writing plan, but the user needed a cleaner file that looks like the thesis itself. The new document should contain formal thesis structure, headings, empty writing space, an updated table of contents, lists of tables and figures, references, and appendices without visible planning instructions in the thesis body.

### Issue

A thesis skeleton should not read like a project plan. Visible notes such as planned length, writing instructions, or draft-space reminders make the document feel like preparation material rather than the actual thesis file.

### Fix

A separate clean skeleton was generated:

1. `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`
2. `docs/thesis/build_thesis_skeleton_docx.py`

The document includes:

1. Formal title page.
2. Abstract page.
3. Updated Word table of contents.
4. List of tables.
5. List of figures.
6. Numbered thesis chapters and subchapters.
7. Empty writing space under headings.
8. References generated from `docs/thesis/references.bib`.
9. Appendices for screenshots, evaluation prompts, benchmarks, configuration examples, and development logs.

The skeleton follows the university formatting rules: A4, Times New Roman 12 pt body text, 1.5 line spacing, 3.5 cm left margin, 2.5 cm other margins, bold uppercase 14 pt chapter headings, and bold 12 pt subchapter headings.

### Verification

The file was opened through Microsoft Word to update the table of contents and exported to PDF for structural checking. The exported PDF contained 37 pages, no visible planning text, and the expected thesis sections and references. LibreOffice-based PNG rendering was still unavailable in the environment.

### Thesis-Relevant Lesson

The clean skeleton can now be used as the working thesis document. Future writing sessions can fill section content directly under the headings while keeping the structure and formatting stable.

### Files Changed

1. `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`
2. `docs/thesis/build_thesis_skeleton_docx.py`
3. `docs/DEVELOPMENT_LOG.md`

## 2026-05-23 - Chapter 1 rewritten for natural thesis style

### Context

The user asked to rewrite the introduction so it sounds more natural and human-written while still following academic and university formatting rules. The user also reminded that references should follow the thesis rule requiring alphabetical ordering.

### Issue

The first version of Chapter 1 was structurally correct, but some phrasing sounded too template-like. The text needed a smoother academic rhythm, a more natural explanation of the author's practical motivation, and continued citation support without overloading the introduction.

### Fix

Chapter 1 was rewritten in `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`. The revised version keeps the same sections:

1. `1 INTRODUCTION`
2. `1.1 Background and motivation`
3. `1.2 Research problem`
4. `1.3 Aim, scope, and contribution`
5. `1.4 Research questions and hypotheses`
6. `1.5 Thesis structure`

The rewrite makes the motivation more natural by explaining API dependency problems through normal software team situations and then connecting them to the author's backend engineering experience. The research problem, aim, scope, contribution, research questions, and thesis structure were also rephrased to avoid a mechanical template style.

The Word footnotes were rebuilt and the table of contents was updated. The generated references remain grouped according to the allowed thesis structure and alphabetized inside each group:

1. Scientific literature.
2. Technical specifications and tool documentation.
3. Additional contextual literature.

### Verification

The document was exported through Microsoft Word to PDF for structural checking. The exported PDF contains 41 pages, five Word footnotes, one updated table of contents, and no visible placeholder markers such as `[[FN1]]`. Formatting checks confirmed A4 page size, Times New Roman 12 pt body text, 1.5 spacing, correct margins, 14 pt chapter headings, and 12 pt subchapter headings.

### Files Changed

1. `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`
2. `docs/thesis/build_thesis_skeleton_docx.py`
3. `docs/DEVELOPMENT_LOG.md`

## 2026-05-30 - Unused thesis references pruned and numeric citations renumbered

### Context

The reference audit showed that the thesis had 96 numbered references, but only 49 were actually cited in the thesis text. The user asked whether the uncited references could be used and whether unnecessary ones should be deleted with corrected numbering.

### Issue

Keeping uncited papers in a final thesis bibliography can look careless because the reference list no longer reflects the evidence actually used in the text. Removing unused entries is not enough by itself, because all numeric citations and internal reference bookmarks must also be renumbered consistently.

### Fix

Added `docs/thesis/prune_unused_references.py`, a local DOCX maintenance script that:

1. Reads the current thesis integrity report logic.
2. Keeps only references that are cited at least once in the thesis body.
3. Preserves the alphabetical order of the remaining bibliography.
4. Renumbers the kept references from `[1]` onward.
5. Updates in-text hyperlink citations so they point to the new reference bookmarks.
6. Removes unused bibliography paragraphs from the `REFERENCES` section.
7. Adds a backup DOCX before modifying the thesis.
8. Inserts a blank page before `REFERENCES` when needed so the section can start on an odd-numbered page according to the university rule.

### Thesis-Relevant Lesson

The bibliography was reduced from a broad candidate-reading list to an actual thesis reference list. This improves academic consistency: every final bibliography item now has a visible role in the thesis text, and the numeric citation system can be checked mechanically.

### Verification

The integrity checker was run after pruning and after refreshing the Word table of contents/exported PDF. Final result:

1. 49 references remain.
2. 49 references are cited at least once.
3. 0 unused references remain.
4. 0 missing citation targets were found.
5. 0 duplicate reference numbers were found.
6. The reference list remains alphabetically ordered.
7. The thesis exports to 84 pages, inside the 70-90 page Master thesis range.
8. The `REFERENCES` section now starts on visible page 77, satisfying the odd-page start rule.

### Files Changed

1. `docs/thesis/prune_unused_references.py`
2. `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`
3. `docs/thesis/thesis_integrity_report.md`
4. `docs/thesis/thesis_integrity_report.json`
5. `docs/DEVELOPMENT_LOG.md`

## 2026-05-30 - Thesis reference and layout integrity checker added

### Context

The user needed a repeatable way to check whether numeric references used in the thesis body are consistent with the final reference list. The same request also raised thesis-format concerns: appendix meaning, remaining screenshot placeholders, margins, body text alignment, and text that appears shifted to the right in Word.

### Issue

Manual citation checking is risky in a long thesis because unused bibliography entries, missing numeric targets, and formatting drift can be missed easily. The thesis also contains official front-matter templates and generated content, so a practical audit needs to distinguish body-text issues from intentionally positioned declaration/title-page content.

### Fix

Added `docs/thesis/check_thesis_integrity.py`, a local Python checker that:

1. Parses the thesis DOCX.
2. Finds the `REFERENCES` section and extracts numbered bibliography entries.
3. Scans the non-reference thesis text for numeric citations such as `[4]` and `[4, 23, 47]`.
4. Reports cited, unused, missing, duplicated, and malformed references.
5. Checks page geometry against the university rules: A4, 3.5 cm left margin, 2.5 cm top/right/bottom margins.
6. Audits body paragraph justification, line spacing, suspicious left indents, direct font overrides, heading conventions, appendices, and remaining visual placeholders.
7. Checks whether required major sections start on odd-numbered pages according to the table of contents and exported PDF.
8. Checks whether appendix headings contain actual text or image content.
9. Writes both Markdown and JSON reports for future thesis-writing work.

Added `docs/thesis/run_thesis_integrity_check.ps1` as a Windows-friendly wrapper that can run the checker with a normal Python installation or the bundled Codex Python runtime.

### Thesis-Relevant Lesson

The thesis process now includes a reproducible quality-control step for citation integrity and formal layout compliance. This is useful evidence for the writing process because it shows that the bibliography and document formatting were not managed only manually, but checked with a purpose-built audit script.

### Files Changed

1. `docs/thesis/check_thesis_integrity.py`
2. `docs/thesis/run_thesis_integrity_check.ps1`
3. `docs/thesis/thesis_integrity_report.md`
4. `docs/thesis/thesis_integrity_report.json`
5. `docs/DEVELOPMENT_LOG.md`

## 2026-05-23 - Chapter 1 introduction drafted

### Context

The user asked to write Chapter 1 of the thesis, covering the introduction, background and motivation, research problem, aim, scope, contribution, research questions, hypotheses, and thesis structure. The user also asked to include personal motivation from professional experience without making the text informal.

### Issue

The introduction needed to sound like a formal Master thesis, not a product pitch or personal blog post. It also needed enough references to ground the problem while leaving the detailed literature review for Chapter 2.

### Fix

Chapter 1 was written directly into `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx` under:

1. `1 INTRODUCTION`
2. `1.1 Background and motivation`
3. `1.2 Research problem`
4. `1.3 Aim, scope, and contribution`
5. `1.4 Research questions and hypotheses`
6. `1.5 Thesis structure`

The motivation section now frames the author's backend engineering experience academically, mentioning work with REST APIs, OpenAPI/Swagger, integration testing, external service integrations, and local development dependency problems.

Five Word footnotes were added using scientific references on:

1. API mocking and mocking-framework usage.
2. JSON Schema and structured output.
3. AI-assisted developer productivity.
4. LLMs for software engineering.
5. Offline/local LLMs.

### Verification

The document was regenerated, citation markers were converted into real Word footnotes, the table of contents was updated, and the document was exported through Microsoft Word to PDF for structural checking. The saved DOCX keeps the required formatting: A4, Times New Roman 12 pt body, 1.5 line spacing, 3.5 cm left margin, 2.5 cm other margins, 14 pt bold chapter headings, and 12 pt bold subchapter headings.

### Files Changed

1. `docs/thesis/GoFaux_Master_Thesis_Skeleton.docx`
2. `docs/thesis/build_thesis_skeleton_docx.py`
3. `docs/DEVELOPMENT_LOG.md`

## 2026-05-20 - Consensus reference export merged and curated

### Context

The user provided a larger BibTeX export from Consensus containing additional papers about LLM code generation, structured output, REST API generation, automated testing, prompt robustness, local/small models, agentic systems, healthcare, cybersecurity, and other broad AI domains.

### Issue

The export contained duplicate works, unstable BibTeX keys, broad domain papers that could distract from the thesis topic, and some entries whose metadata needs final verification. A direct paste would make the bibliography look large but less academically focused.

### Fix

The thesis bibliography was expanded with cleaned citation keys and deduplicated entries. Core references were added for:

1. LLM-assisted REST API documentation and specification generation.
2. Structured JSON output and JSON Schema benchmarks.
3. LLM evaluation methodology.
4. LLM-assisted software testing and test-case generation.
5. Prompt engineering, robustness, and secure generation.
6. Small/local model trade-offs.
7. Broader software-engineering surveys.

Broad healthcare, cybersecurity, graph, text-to-SQL, RAG, and agentic-system references were preserved as background-only candidates and marked in the reference map so future thesis-writing work can avoid overusing them.

### Thesis-Relevant Lesson

The bibliography now supports a stronger scientific framing for GoFaux: the thesis can connect its implementation to API mocking, REST testing, OpenAPI contracts, JSON Schema validation, local LLM deployment, structured output evaluation, and LLM reliability.

Thesis-ready sentence:

> The reference set was curated rather than merely expanded: duplicate and weakly related entries were separated from core sources so that the literature review can stay focused on local AI-assisted mock API generation, schema-constrained JSON output, and empirical evaluation of model reliability.

### Files Changed

1. `docs/thesis/references.bib`
2. `docs/thesis/REFERENCE_MAP.md`
3. `docs/DEVELOPMENT_LOG.md`
