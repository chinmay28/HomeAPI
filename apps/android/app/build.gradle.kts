plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
}

// Full-stack mode runs the real homeapi binary, built from this repository by
// the Go toolchain and packaged as a native library (libhomeapi.so), which is
// the one place Android lets an app keep a file it may execute.
//
//   -PappServer=false             build a thin client only, without Go
//   -PappServerAbis=arm64-v8a,x86_64
//                                 which ABIs get a server (x86_64 is for the
//                                 emulator); arm64-v8a by default
//   -PappGo=/path/to/go           the go command, if it is not on PATH
//   -PappVersionName=v2026.11.30 -PappVersionCode=30 -PappVersionDate=2026-11
//                                 stamped by the release workflow from
//                                 scripts/version.mjs, so the APK, the
//                                 server inside it and its web GUI agree
val repoRoot = rootProject.projectDir.resolve("../..").canonicalFile
val withServer = (findProperty("appServer") as String?) != "false"
val serverAbis = ((findProperty("appServerAbis") as String?) ?: "arm64-v8a")
    .split(',').map { it.trim() }.filter { it.isNotEmpty() }
val goCommand = (findProperty("appGo") as String?) ?: "go"
val serverLibs = layout.buildDirectory.dir("generated/homeapiServer/jniLibs")

// A static linux binary, not GOOS=android: HomeAPI's SQLite is
// modernc.org/sqlite, whose C-to-Go libc targets linux (CountRoster learned
// the same, MOBILE.md §2.1), and CGO_ENABLED=0 linux/arm64 runs on Android
// as it stands. The zone database is built in (timetzdata), since a phone has
// none Go can read; HOMEAPI_TZ names the zone.
data class GoTarget(val goos: String, val goarch: String)
val goTargets = mapOf(
    "arm64-v8a" to GoTarget("linux", "arm64"),
    "x86_64" to GoTarget("linux", "amd64"),
)

android {
    namespace = "io.github.chinmay28.homeapi"
    compileSdk = 35

    defaultConfig {
        applicationId = "io.github.chinmay28.homeapi"
        // Android 10: scoped storage and MediaStore.Downloads, so saving an
        // export needs no storage permission.
        minSdk = 29
        targetSdk = 35
        // The release's version as the name, its commit count as the code
        // (which only grows, as Android requires of an update).
        versionCode = (findProperty("appVersionCode") as String?)?.toInt() ?: 1
        versionName = (findProperty("appVersionName") as String?) ?: "dev"
        buildConfigField("boolean", "HAS_SERVER", withServer.toString())
    }

    buildFeatures {
        buildConfig = true
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            // Signed with the debug key so the release build sideloads as it
            // stands. Anybody publishing to a store signs it with their own.
            signingConfig = signingConfigs.getByName("debug")
        }
    }

    sourceSets["main"].jniLibs.srcDir(serverLibs)

    packaging {
        jniLibs {
            // Installed as a file on disk, not mapped from inside the APK:
            // the server is exec'd, not loaded.
            useLegacyPackaging = true
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions {
        jvmTarget = "17"
    }
    lint {
        disable += setOf(
            // Plain HTTP is how HomeAPI runs on a home network and on
            // 127.0.0.1, and a private CA is how it runs behind TLS at home:
            // both are the point (see network_security_config.xml).
            "InsecureBaseConfiguration",
            "AcceptsUserCertificates",
            // allowBackup is off; there is nothing for fullBackupContent to say.
            "DataExtractionRules",
            // Versions move on purpose, together, not whenever lint notices.
            "GradleDependency",
        )
    }
}

if (withServer) {
    val frontendBuild = repoRoot.resolve("cmd/homeapi/frontend_build")
    // The binary embeds the web GUI; without a real build in
    // cmd/homeapi/frontend_build it would serve a blank page, so build one
    // when it is missing (`make frontend`, which needs Node) and refuse to go
    // on if that leaves nothing there.
    val frontend = tasks.register<Exec>("buildHomeapiFrontend") {
        description = "Builds the web GUI into cmd/homeapi/frontend_build (make frontend)"
        workingDir = repoRoot
        onlyIf { !frontendBuild.resolve("static").isDirectory }
        commandLine("make", "frontend")
    }
    val checkFrontend = tasks.register("checkHomeapiFrontend") {
        dependsOn(frontend)
        doLast {
            if (!frontendBuild.resolve("index.html").isFile || !frontendBuild.resolve("static").isDirectory) {
                throw GradleException("no web GUI build in $frontendBuild — run `make frontend` in the repository root")
            }
        }
    }

    val version = (findProperty("appVersionCode") as String?) ?: "0"
    val date = (findProperty("appVersionDate") as String?) ?: ""
    val versionPkg = "github.com/chinmay28/homeapi/internal/version"
    val buildAll = tasks.register("buildHomeapiServer")
    for (abi in serverAbis) {
        val target = goTargets[abi] ?: throw GradleException("no Go target for ABI $abi (known: ${goTargets.keys})")
        val out = serverLibs.map { it.file("$abi/libhomeapi.so") }
        val task = tasks.register<Exec>("buildHomeapiServer-$abi") {
            description = "Builds homeapi for $abi as libhomeapi.so"
            dependsOn(checkFrontend)
            workingDir = repoRoot
            inputs.files(fileTree(repoRoot) {
                include("go.mod", "go.sum", "cmd/homeapi/**", "internal/**")
                exclude("**/*_test.go", "**/testdata/**")
            })
            inputs.property("version", version)
            inputs.property("date", date)
            outputs.file(out)
            environment("CGO_ENABLED", "0")
            environment("GOOS", target.goos)
            environment("GOARCH", target.goarch)
            commandLine(
                goCommand, "build", "-trimpath", "-tags", "timetzdata",
                "-ldflags=-s -w -X $versionPkg.Patch=$version -X $versionPkg.Date=$date",
                "-o", out.get().asFile.absolutePath,
                "./cmd/homeapi",
            )
        }
        buildAll.configure { dependsOn(task) }
    }
    tasks.named("preBuild") { dependsOn(buildAll) }
}

dependencies {
    implementation(project(":core"))
    implementation("androidx.core:core-ktx:1.13.1")
    implementation("androidx.activity:activity-ktx:1.9.3")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("androidx.preference:preference-ktx:1.2.1")
    implementation("androidx.webkit:webkit:1.12.1")
    implementation("com.google.android.material:material:1.12.0")
}
