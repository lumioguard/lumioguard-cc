package cfamily

import (
	"strings"
	"testing"

	"github.com/lumioguard/lumioguard-cc/internal/adapter"
	"github.com/lumioguard/lumioguard-cc/internal/adapter/adaptertest"
	"github.com/lumioguard/lumioguard-cc/internal/domain"
)

func analyze(t *testing.T, name, code string) *adapter.SourceFile {
	t.Helper()
	file := Analyze(name, "/repo/"+name, code)
	if len(file.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", file.Diagnostics)
	}
	return file
}

var (
	expect        = adaptertest.Expect
	expectSymbols = adaptertest.ExpectSymbols
	value         = adaptertest.Value
)

func TestReferenceFixture(t *testing.T) {
	for _, name := range []string{"sample.c", "sample.cpp"} {
		file := analyze(t, name, `#include <stdbool.h>

int score(bool a, bool b, int c) {
    if (a && b) {
        for (int i = 0; i < 2; i++) {
            if (c > 0) {
                return i;
            }
        }
    }
    return 0;
}
`)
		expect(t, file, domain.MetricCyclomatic, "score", 5)
		expect(t, file, domain.MetricNestingDepth, "score", 3)
		expect(t, file, domain.MetricParameterCount, "score", 3)
		expect(t, file, domain.MetricCognitive, "score", 7)
		expect(t, file, domain.MetricFunctionLines, "score", 10)
	}
}

func TestCppDeclarationsAndNaming(t *testing.T) {
	file := analyze(t, "names.cpp", `namespace app::detail {
template <typename T, int N = (3 > 2)>
class [[nodiscard]] Store final : public Base<T> {
    Q_OBJECT
public:
    Store() : items_{}, size_(0), Base<T>(1) { init(); }
    ~Store() override {}
    Store(const Store&) = delete;
    bool operator==(const Store& other) const noexcept { return size_ == other.size_; }
    int operator()(int a, int b) { return a + b; }
    void declared(int x);
public slots:
    void tick() ABSL_LOCKS_EXCLUDED(mu_) {}
private:
    struct Node { int value() const { return 1; } };
    std::map<int, int> items_;
};

template <class T>
void Store<T>::declared(int x) {
    auto sorter = [this](int a, int b) -> bool { return a < b; };
    run([&] { if (x) { return; } });
}

extern "C" {
DLL_EXPORT int __stdcall c_entry(void) { return 0; }
}
}  // namespace app::detail

static std::vector<int> make(std::pair<int, int> p, ...) try {
    return {};
} catch (const std::exception& e) {
    return {};
}
`)
	expectSymbols(t, file,
		"Store.Store", "Store.~Store", "Store.operator==", "Store.operator()", "Store.tick", "Node.value",
		"Store.declared", "Store.declared.sorter", "Store.declared.<lambda>", "c_entry", "make")
	expect(t, file, domain.MetricParameterCount, "Store.operator()", 2)
	expect(t, file, domain.MetricParameterCount, "c_entry", 0)
	expect(t, file, domain.MetricParameterCount, "make", 2)
	expect(t, file, domain.MetricParameterCount, "Store.declared.sorter", 2)
	// The anonymous lambda's if counts in its own function and, nested, in the method's cognitive total.
	expect(t, file, domain.MetricCyclomatic, "Store.declared.<lambda>", 2)
	expect(t, file, domain.MetricCyclomatic, "Store.declared", 1)
	expect(t, file, domain.MetricCognitive, "Store.declared", 2)
	expect(t, file, domain.MetricCyclomatic, "make", 2)
}

func TestBranchesJumpsAndConditionals(t *testing.T) {
	file := analyze(t, "branches.c", `int branches(int a, int b) {
    if (a > 0) {
        return 1;
    } else if (b > 0) {
        return 2;
    } else {
        if (a == b) return 3;
    }
    switch (a) {
    case 1:
    case 2: a++; break;
    case 3: { a--; } break;
    default: break;
    }
    while (a--) { if (b) continue; }
    do { b++; } while (b < 10);
    if (a) goto done;
    return a ? (b ? 1 : 2) : 3;
done:
    return 0;
}
`)
	// 1 + if, else-if, nested if, 3 non-default cases, while, if, do, if, 2 ternaries = 13
	expect(t, file, domain.MetricCyclomatic, "branches", 13)
	// if 1, else-if 1, else 1, nested if 2, switch 1, while 1, if 2, do 1, if 1, goto 1,
	// ternary 1, nested ternary 2 = 15
	expect(t, file, domain.MetricCognitive, "branches", 15)
	// An else-if nests inside its if, so the if in the final else is at depth 3.
	expect(t, file, domain.MetricNestingDepth, "branches", 3)
}

func TestLogicalOperatorsAndReferences(t *testing.T) {
	file := analyze(t, "logic.cpp", `bool ok(bool a, bool b, bool c, bool d) {
    return a && b && (c || d);
}
bool spelled(bool a, bool b, bool c) {
    return a and b or c;
}
void refs(std::vector<int>&& v, auto&&... rest) {
    auto&& x = v;
    std::string&& s = make();
    for (auto&& item : v) {}
    for (Item&& item : v) {}
    call(static_cast<T&&>(x), [](auto&& y) { return y; });
}
`)
	expect(t, file, domain.MetricCyclomatic, "ok", 4)
	expect(t, file, domain.MetricCognitive, "ok", 2)
	expect(t, file, domain.MetricCyclomatic, "spelled", 3)
	expect(t, file, domain.MetricCognitive, "spelled", 2)
	// Two loops and no logical operators.
	expect(t, file, domain.MetricCyclomatic, "refs", 3)
	expect(t, file, domain.MetricParameterCount, "refs", 2)
}

func TestAlternativeSpellingsAreIdentifiersInC(t *testing.T) {
	file := analyze(t, "names.c", `int f(int and, int or) {
    int new = and + or;
    return new;
}
`)
	expect(t, file, domain.MetricCyclomatic, "f", 1)
}

func TestPreprocessorAndMacros(t *testing.T) {
	file := analyze(t, "kernel.c", `#define FOR_EACH(i, n) for (i = 0; i < n; i++)
static int SYSCALL_DEFINE2(count, int, n) {
    int i, total = 0;
#ifdef FAST
    if (n > 100) {
#else
    if (n > 10) {
#endif
        total = n;
    }
#if 0
    if (never) {
#endif
    list_for_each(i, n) {
        if (i % 2) total++;
    }
    FOR_EACH(i, n) if (i) total--;
    return total;
}
`)
	expectSymbols(t, file, "SYSCALL_DEFINE2")
	expect(t, file, domain.MetricParameterCount, "SYSCALL_DEFINE2", 3)
	// 1 + the kept if + the if in the macro block + the if after the macro
	expect(t, file, domain.MetricCyclomatic, "SYSCALL_DEFINE2", 4)
}

func TestMacrosAroundConditions(t *testing.T) {
	file := analyze(t, "macros.cpp", `void run(const char *s) {
    if FMT_CONSTEXPR20 (s) { go(); }
    if EQ("x") return;
    if consteval { fast(); } else { slow(); }
    IF_THREW {
        clear();
    } else if (s) {
        next();
    }
    struct itimerval timer {
        {1, 0}, {1, 0}
    };
    struct Local { int twice(int x) { return x ? 2 * x : 0; } };
}
`)
	expectSymbols(t, file, "run", "Local.run.twice")
	// 1 + three ifs + the macro that takes an else + its else-if
	expect(t, file, domain.MetricCyclomatic, "run", 6)
	expect(t, file, domain.MetricCyclomatic, "Local.run.twice", 2)
}

func TestMacrosBeforeDeclarations(t *testing.T) {
	file := analyze(t, "absl.cc", `namespace absl {
ABSL_NAMESPACE_BEGIN
namespace internal {
class ABSL_SCOPED_LOCKABLE Lock {
 public:
  explicit Lock(Arena *arena) ABSL_EXCLUSIVE_LOCK_FUNCTION(arena->mu) : arena_(arena) { arena_->mu.lock(); }
 private:
  Arena *arena_;
};
}  // namespace internal
ABSL_NAMESPACE_END
}  // namespace absl
`)
	expectSymbols(t, file, "Lock.Lock")
}

func TestHeadersHoldingCpp(t *testing.T) {
	file := analyze(t, "memory.h", `template <class T, class F>
struct Impl {
  template <class... Args>
  decltype(std::declval<F>()(std::declval<T>())) operator()(Args&&... args) const { return 1; }
};
JSON_DEPRECATED_FOR(3.11.0, to_string())
operator string_t() const { return ""; }
`)
	expectSymbols(t, file, "Impl.operator()", "operator string_t")
}

func TestNamesBuiltOrShieldedByParentheses(t *testing.T) {
	file := analyze(t, "wrapped.c", `LUALIB_API lua_State *(luaL_newstate) (void) { return 0; }
ABSL_ATTRIBUTE_WEAK void ABSL_INTERNAL_C_SYMBOL(AbslYield)() {}
WRAP_INT_DB(1changes, sqlite3_changes)
WRAP_INT_DB(1total, sqlite3_total)
S3JniApi(sqlite3_context(),jlong,1context)(JNIEnv *env, jobject cx){
  return 0;
}
local int try(char *hex, int err) { return err; }
`)
	expectSymbols(t, file, "luaL_newstate", "AbslYield", "S3JniApi", "try")
	expect(t, file, domain.MetricParameterCount, "S3JniApi", 2)
	if got := value(t, file, domain.MetricFunctionLines, "S3JniApi"); got != 3 {
		t.Errorf("the function must start after the macro chain, got %v lines", got)
	}
}

func TestMacrosInsideConstructorInitializers(t *testing.T) {
	file := analyze(t, "impl.cc", `Impl::Impl(Parent* parent)
    : parent_(parent),
      DISABLE_WARNINGS_PUSH_(4355)
          reporter_(this),
      DISABLE_WARNINGS_POP_() other_(&reporter_),
      flag_(false) {
  if (parent_) init();
}
`)
	expectSymbols(t, file, "Impl.Impl")
	expect(t, file, domain.MetricCyclomatic, "Impl.Impl", 2)
}

func TestUnrecognisedBlocksFailInsteadOfVanishing(t *testing.T) {
	file := Analyze("hidden.c", "/repo/hidden.c", "BEGIN_BLOCK {\n  int f(void) { return 1; }\n}\n")
	if !file.HasRequiredDiagnostic() || !strings.Contains(file.Diagnostics[0].Message, "unrecognised declaration") {
		t.Fatalf("diagnostics = %+v", file.Diagnostics)
	}
}

func TestClassicCDeclarations(t *testing.T) {
	file := analyze(t, "classic.c", `typedef struct point { int x, y; } point;
static const int table[] = { 1, 2, 3 };
struct point *origin(void) { static struct point p = { .x = 0 }; return &p; }
void (*signal(int sig, void (*handler)(int)))(int) { return handler; }
int __attribute__((noinline)) attributed(const char *restrict s) { return s[0]; }
int apply(int (*fn)(int, int), int a) { return fn(a, a); }
enum color { RED = (1 << 0), GREEN };
int declared(int);
`)
	expectSymbols(t, file, "origin", "signal", "attributed", "apply")
	expect(t, file, domain.MetricParameterCount, "signal", 2)
	expect(t, file, domain.MetricParameterCount, "apply", 2)
}

func TestFunctionLinesExcludeCommentsAndBlankLines(t *testing.T) {
	file := analyze(t, "lines.c", `/* Documented above, so outside the function. */
int lines(void) {
    // explanation

    int value = 1; /* inline */
    const char *text = "a /* not a comment */";
    return value; // trailing
}
`)
	expect(t, file, domain.MetricFunctionLines, "lines", 5)
}

func TestTokensAndIncludes(t *testing.T) {
	file := analyze(t, "tokens.c", `#include "local.h" // why
#include <sys/types.h>
#include CONFIG_HEADER
/* comment */
int x = 1;
`)
	var values []string
	for _, token := range file.Tokens {
		values = append(values, token.Value)
	}
	want := `# include "local.h" # include <sys/types.h> # include CONFIG_HEADER int x = 1 ;`
	if got := strings.Join(values, " "); got != want {
		t.Fatalf("tokens = %q", got)
	}
	if len(file.Imports) != 2 || file.Imports[0].Specifier != `"local.h"` || file.Imports[1].Specifier != "<sys/types.h>" {
		t.Fatalf("imports = %+v", file.Imports)
	}
	if len(file.ImportSpans) != 3 || file.ImportSpans[2].Line != 3 {
		t.Fatalf("spans = %+v", file.ImportSpans)
	}
}

func TestFailuresAreRequiredDiagnostics(t *testing.T) {
	cases := map[string]string{
		"open.c":    "void f() {\n  if (x) {\n}\n",
		"stray.c":   "void f() { }\n}\n",
		"else.c":    "void f() { else x(); }\n",
		"cond.c":    "#ifdef A\nvoid f() {}\n",
		"objc.h":    "@interface Foo : NSObject\n@end\n",
		"kandr.c":   "int f(a) int a; { return a; }\n",
		"literal.c": "const char *s = \"open;\n",
	}
	for name, code := range cases {
		file := Analyze(name, "/repo/"+name, code)
		if !file.HasRequiredDiagnostic() || file.Diagnostics[0].Code != "cfamily.parse_failed" || len(file.Measurements) != 0 {
			t.Errorf("%s: diagnostics %+v measurements %d", name, file.Diagnostics, len(file.Measurements))
		}
	}
}
