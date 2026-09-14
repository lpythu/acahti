package forgejo

import "testing"

func TestParseUnifiedDiff(t *testing.T) {
	raw := `diff --git a/old.txt b/new.txt
rename from old.txt
rename to new.txt
--- a/old.txt
+++ b/new.txt
@@ -1,3 +1,3 @@
 keep
-old
+new
diff --git a/gone.txt b/gone.txt
deleted file mode 100644
--- a/gone.txt
+++ /dev/null
@@ -1 +0,0 @@
-bye
diff --git a/fresh.txt b/fresh.txt
new file mode 100644
--- /dev/null
+++ b/fresh.txt
@@ -0,0 +1 @@
+hi
`
	files := ParseUnifiedDiff(raw)
	if len(files) != 3 {
		t.Fatalf("files=%d", len(files))
	}
	if files[0].Filename != "new.txt" || files[0].Status != "renamed" || files[0].PreviousFilename != "old.txt" {
		t.Fatalf("rename %+v", files[0])
	}
	if files[0].Additions != 1 || files[0].Deletions != 1 {
		t.Fatalf("rename counts %+v", files[0])
	}
	if files[1].Filename != "gone.txt" || files[1].Status != "removed" || files[1].Deletions != 1 {
		t.Fatalf("removed %+v", files[1])
	}
	if files[2].Filename != "fresh.txt" || files[2].Status != "added" || files[2].Additions != 1 {
		t.Fatalf("added %+v", files[2])
	}
}
