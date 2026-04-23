package jvm

import (
	"reflect"
	"testing"
)

func TestSplitArgsAddOpensPair(t *testing.T) {
	args := []string{
		"--add-opens", "java.base/java.lang=ALL-UNNAMED",
		"-jar", "game.jar", "--foo",
	}
	jvm, main, app := splitArgs(args)
	wantJVM := []string{"--add-opens", "java.base/java.lang=ALL-UNNAMED", "-jar", "game.jar"}
	if !reflect.DeepEqual(jvm, wantJVM) {
		t.Fatalf("jvm args: got %#v want %#v", jvm, wantJVM)
	}
	if main != "" {
		t.Fatalf("mainClass: got %q want empty for -jar", main)
	}
	if len(app) != 1 || app[0] != "--foo" {
		t.Fatalf("app: got %#v want [--foo]", app)
	}
}

func TestSplitArgsAddOpensThenMainClass(t *testing.T) {
	args := []string{
		"--add-opens", "java.base/java.lang=ALL-UNNAMED",
		"-cp", "lib.jar", "com.game.Main", "run",
	}
	jvm, main, app := splitArgs(args)
	wantJVM := []string{"--add-opens", "java.base/java.lang=ALL-UNNAMED", "-cp", "lib.jar"}
	if !reflect.DeepEqual(jvm, wantJVM) {
		t.Fatalf("jvm: %#v", jvm)
	}
	if main != "com.game.Main" || len(app) != 1 || app[0] != "run" {
		t.Fatalf("main=%q app=%#v", main, app)
	}
}

func TestSplitArgsClasspathUnchanged(t *testing.T) {
	args := []string{"-cp", "a.jar", "com.example.Main", "arg1"}
	jvm, main, app := splitArgs(args)
	if !reflect.DeepEqual(jvm, []string{"-cp", "a.jar"}) {
		t.Fatalf("jvm: %#v", jvm)
	}
	if main != "com.example.Main" || len(app) != 1 || app[0] != "arg1" {
		t.Fatalf("main=%q app=%#v", main, app)
	}
}

func TestIsLikelyGameLaunch(t *testing.T) {
	bootstrap := []string{
		"-Djdk.attach.allowAttachSelf=true",
		"-cp", "Launcher.jar",
		"pro.gravit.launcher.LauncherEngineWrapper",
	}
	if IsLikelyGameLaunch(bootstrap) {
		t.Fatal("gravit bootstrap must not be treated as game launch")
	}

	game := []string{
		"-cp", "client.jar",
		"pro.gravit.launcher.huITAXZAEVkrQX",
		"--gameDir", "C:/game",
		"--assetsDir", "C:/assets",
	}
	if !IsLikelyGameLaunch(game) {
		t.Fatal("game argv markers must enable tuning")
	}
}

func TestFilterArgsStripsLegacyAndG1Flags(t *testing.T) {
	orig := []string{
		"--illegal-access=warn",
		"-XX:+UseConcMarkSweepGC",
		"-XX:MaxPermSize=256m",
		"-XX:+UseG1GC",
		"-XX:MaxGCPauseMillis=200",
		"-Xmx4g",
		"--add-opens", "java.base/java.lang=ALL-UNNAMED",
		"-cp", "lib.jar",
		"com.example.Main", "--gameDir", "C:/game",
	}
	injected := []string{"-XX:+UseZGC", "-Xmx8g"}
	got := FilterArgs(orig, injected)

	for _, bad := range []string{
		"-XX:+UseConcMarkSweepGC",
		"-XX:MaxPermSize=256m",
		"-XX:+UseG1GC",
		"-XX:MaxGCPauseMillis=200",
		"-Xmx4g",
		"--illegal-access=warn",
	} {
		for _, g := range got {
			if g == bad {
				t.Fatalf("FilterArgs kept legacy flag %q", bad)
			}
		}
	}

	for _, want := range []string{"-XX:+UseZGC", "-Xmx8g", "com.example.Main", "--gameDir"} {
		found := false
		for _, g := range got {
			if g == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("FilterArgs dropped required arg %q: %v", want, got)
		}
	}
}
