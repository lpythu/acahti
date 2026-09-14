import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom"

import { TooltipProvider } from "@/components/ui/tooltip"
import { I18nProvider } from "@/i18n/i18n"
import { SessionProvider } from "@/lib/session"
import { AdminHomePage } from "@/pages/admin-home"
import { BoardPage } from "@/pages/board"
import { KeysPage } from "@/pages/keys"
import { LoginPage } from "@/pages/login"
import { PackageDetailPage } from "@/pages/package-detail"
import { PackagesPage } from "@/pages/packages"
import { PipelinePage } from "@/pages/pipeline"
import { PipelinesPage } from "@/pages/pipelines"
import { PullPage } from "@/pages/pull"
import { RepoBranchesPage } from "@/pages/repo-branches"
import { RepoCommitPage } from "@/pages/repo-commit"
import { RepoCommitsPage } from "@/pages/repo-commits"
import { RepoFilesPage } from "@/pages/repo-files"
import { RepoLayout } from "@/pages/repo-layout"
import { RepoPipelinesPage } from "@/pages/repo-pipelines"
import { RepoAccessPage } from "@/pages/repo-access"
import { RepoPullsPage } from "@/pages/repo-pulls"
import { ReposPage } from "@/pages/repos"
import { AdminGate, AppShell } from "@/pages/shell"
import { ConsentPage } from "@/pages/consent"
import { JoinPage } from "@/pages/join"
import { UsePage } from "@/pages/use"
import { UsersPage } from "@/pages/users"

export default function App() {
  return (
    <I18nProvider>
      <TooltipProvider>
        <BrowserRouter>
          <SessionProvider>
            <Routes>
              <Route path="/" element={<UsePage />} />
              <Route path="/login" element={<LoginPage />} />
              <Route path="/join" element={<JoinPage />} />
              <Route path="/oauth/consent" element={<ConsentPage />} />
              <Route element={<AppShell />}>
                <Route path="/board" element={<BoardPage />} />
                <Route path="/repos" element={<ReposPage />} />
                <Route path="/repos/:owner/:name" element={<RepoLayout />}>
                  <Route index element={<RepoFilesPage />} />
                  <Route path="commits" element={<RepoCommitsPage />} />
                  <Route path="commits/:sha" element={<RepoCommitPage />} />
                  <Route path="branches" element={<RepoBranchesPage />} />
                  <Route path="pulls" element={<RepoPullsPage />} />
                  <Route path="pulls/:number" element={<PullPage />} />
                  <Route path="pipelines" element={<RepoPipelinesPage />} />
                  <Route path="access" element={<RepoAccessPage />} />
                </Route>
                <Route path="/pipelines" element={<PipelinesPage />} />
                <Route path="/pipelines/:owner/:name/:number" element={<PipelinePage />} />
                <Route path="/packages" element={<PackagesPage />} />
                <Route path="/packages/:kind/*" element={<PackageDetailPage />} />
                <Route path="/account/keys" element={<KeysPage />} />
                <Route element={<AdminGate />}>
                  <Route path="/admin" element={<AdminHomePage />} />
                  <Route path="/admin/users" element={<UsersPage />} />
                </Route>
              </Route>
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </SessionProvider>
        </BrowserRouter>
      </TooltipProvider>
    </I18nProvider>
  )
}
