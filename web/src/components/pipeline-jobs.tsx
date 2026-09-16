import { useState } from "react"
import { ChevronRightIcon, FileIcon } from "lucide-react"
import { cn } from "cn"

import { RunStatusIcon } from "@/components/run-status-icon"
import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible"
import {
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
} from "@/components/ui/sidebar"
import { useT } from "@/i18n/i18n"
import type { FileBlob, Step } from "@/lib/api"
import type { Job } from "@/lib/pipeline"

function JobNode({
  job,
  activeStep,
  activeFile,
  onStep,
}: {
  job: Job
  activeStep?: Step | null
  activeFile?: FileBlob | null
  onStep: (step: Step) => void
}) {
  const [open, setOpen] = useState(true)

  return (
    <SidebarMenuItem>
      <Collapsible open={open} onOpenChange={setOpen}>
        <SidebarMenuButton tooltip={job.name} onClick={() => setOpen(!open)}>
          <ChevronRightIcon className={cn("size-4 transition-transform", open && "rotate-90")} />
          <RunStatusIcon status={job.state} wait={job.wait} />
          <span>{job.name}</span>
        </SidebarMenuButton>
        <CollapsibleContent>
          <SidebarMenuSub>
            {job.steps.map((s) => {
              const selected = !activeFile && activeStep?.pid === s.pid && activeStep.name === s.name
              const state = s.state
              return (
                <SidebarMenuSubItem key={`${s.pid}-${s.name}`}>
                  <SidebarMenuSubButton
                    isActive={selected}
                    render={
                      <button
                        type="button"
                        onClick={() => onStep(s)}
                      />
                    }
                  >
                    <RunStatusIcon status={state} wait={state === "pending" ? job.wait : undefined} />
                    <span>{s.name || `#${s.pid}`}</span>
                  </SidebarMenuSubButton>
                </SidebarMenuSubItem>
              )
            })}
          </SidebarMenuSub>
        </CollapsibleContent>
      </Collapsible>
    </SidebarMenuItem>
  )
}

export function PipelineJobs({
  jobs,
  files,
  activeStep,
  activeFile,
  onStep,
  onFile,
}: {
  jobs: Job[]
  files: FileBlob[]
  activeStep?: Step | null
  activeFile?: FileBlob | null
  onStep: (step: Step) => void
  onFile: (file: FileBlob) => void
}) {
  const t = useT()

  return (
    <div className="flex flex-col">
      <SidebarGroup>
        <SidebarGroupLabel>{t("jobs")}</SidebarGroupLabel>
        <SidebarGroupContent>
          <SidebarMenu>
            {jobs.map((job) => (
              <JobNode
                key={job.name}
                job={job}
                activeStep={activeStep}
                activeFile={activeFile}
                onStep={onStep}
              />
            ))}
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
      <SidebarGroup>
        <SidebarGroupLabel>{t("pipelineFiles")}</SidebarGroupLabel>
        <SidebarGroupContent>
          {files.length ? (
            <SidebarMenu>
              {files.map((f) => (
                <SidebarMenuItem key={f.path}>
                  <SidebarMenuButton
                    isActive={activeFile?.path === f.path}
                    tooltip={f.path}
                    render={<button type="button" onClick={() => onFile(f)} />}
                  >
                    <FileIcon />
                    <span>{f.name || f.path}</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          ) : (
            <p className="px-2 text-xs text-muted-foreground">{t("noPipelineFiles")}</p>
          )}
        </SidebarGroupContent>
      </SidebarGroup>
    </div>
  )
}
