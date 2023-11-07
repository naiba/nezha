const mixinsVue = {
    data: {
        cache: [],
        theme: "light",
        isSystemTheme: false
    },
    created() {
        this.initTheme()
    },
    methods: {
        setTheme(title, store = false) {
            this.theme = title
            document.body.setAttribute("theme", title)
            if (store) {
                localStorage.setItem("theme", title)
                this.isSystemTheme = false
            }
        },
        setSystemTheme() {
            localStorage.removeItem("theme")
            this.initTheme()
            this.isSystemTheme = true
        },
        initTheme() {
            const storeTheme = localStorage.getItem("theme")
            if (storeTheme === 'dark' || storeTheme === 'light') {
                this.setTheme(storeTheme, true);
            } else {
                this.isSystemTheme = true
                const handleChange = (mediaQueryListEvent) => {
                    if (localStorage.getItem("theme")) {
                        return
                    }
                    if (mediaQueryListEvent.matches) {
                        this.setTheme('dark');
                    } else {
                        this.setTheme('light');
                    }
                }
                const mediaQueryListDark = window.matchMedia('(prefers-color-scheme: dark)');
                this.setTheme(mediaQueryListDark.matches ? 'dark' : 'light');
                mediaQueryListDark.addEventListener("change", handleChange);
            }
        },
        toFixed2(f) {
            return f.toFixed(2)
        },
    }
}