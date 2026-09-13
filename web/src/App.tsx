import { BrowserRouter, Navigate, Route, Routes, useParams } from "react-router-dom"

import { TooltipProvider } from "@/components/ui/tooltip"
import { Toaster } from "@/components/ui/sonner"
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
import { RepoCommitsPage } from "@/pages/repo-commits"
import { RepoFilesPage } from "@/pages/repo-files"
import { RepoLayout } from "@/pages/repo-layout"
import { RepoPipelinesPage } from "@/pages/repo-pipelines"
import { RepoPullsPage } from "@/pages/repo-pulls"
import { ReposPage } from "@/pages/repos"
import { AdminShell, ConsoleShell } from "@/pages/shell"
import { TokenPage } from "@/pages/token"
import { UsersPage } from "@/pages/users"

function PipelineRedirect() {
  const { owner, name, number } = useParams()
  return <Navigate to={`/repos/${owner}/${name}/pipelines/${number}`} replace />
}

export default function App() {
  return (
    <I18nProvider>
      <TooltipProvider>
        <BrowserRouter>
          <SessionProvider>
            <Routes>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/keys" element={<Navigate to="/account/keys" replace />} />
              <Route path="/token" element={<Navigate to="/account/token" replace />} />
              <Route path="/users" element={<Navigate to="/admin/users" replace />} />
              <Route element={<ConsoleShell />}>
                <Route path="/" element={<BoardPage />} />
                <Route path="/repos" element={<ReposPage />} />
                <Route path="/repos/:owner/:name" element={<RepoLayout />}>
                  <Route index element={<RepoFilesPage />} />
                  <Route path="commits" element={<RepoCommitsPage />} />
                  <Route path="branches" element={<RepoBranchesPage />} />
                  <Route path="pulls" element={<RepoPullsPage />} />
                  <Route path="pulls/:number" element={<PullPage />} />
                  <Route path="pipelines" element={<RepoPipelinesPage />} />
                  <Route path="pipelines/:number" element={<PipelinePage />} />
                </Route>
                <Route path="/pipelines" element={<PipelinesPage />} />
                <Route path="/pipelines/:owner/:name/:number" element={<PipelineRedirect />} />
                <Route path="/packages" element={<PackagesPage />} />
                <Route path="/packages/:kind/:name" element={<PackageDetailPage />} />
                <Route path="/account/keys" element={<KeysPage />} />
                <Route path="/account/token" element={<TokenPage />} />
              </Route>
              <Route element={<AdminShell />}>
                <Route path="/admin" element={<AdminHomePage />} />
                <Route path="/admin/users" element={<UsersPage />} />
              </Route>
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </SessionProvider>
        </BrowserRouter>
        <Toaster />
      </TooltipProvider>
    </I18nProvider>
  )
}
