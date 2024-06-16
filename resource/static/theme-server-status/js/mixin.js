const mixinsVue = {
    data: {
        cache: [],
        theme: "light",
        isSystemTheme: false,
        showGroup: false,
        showGoTop: false,
        preferredTemplate: null,
        isMobile: false,
        adaptedTemplates: [
            { key: 'default', name: 'Default', icon: 'th large' },
            { key: 'angel-kanade', name: 'AngelKanade', icon: 'square' },
            { key: 'server-status', name: 'ServerStatus', icon: 'list' }
        ]
    },
    created() {
        this.isMobile = this.checkIsMobile();
        this.initTheme();
        this.storedShowGroup();
        this.preferredTemplate = this.getCookie('preferred_theme') ? this.getCookie('preferred_theme') : this.$root.defaultTemplate;
        window.addEventListener('scroll', this.handleScroll);
    },
    destroyed() {
        window.removeEventListener('scroll', this.handleScroll);
    },
    methods: {
        toggleView() {
            this.showGroup = !this.showGroup;
            localStorage.setItem("showGroup", JSON.stringify(this.showGroup));
            return this.showGroup;
        },
        storedShowGroup() {
            const storedShowGroup = localStorage.getItem("showGroup");
            if (storedShowGroup !== null) {
                this.showGroup = JSON.parse(storedShowGroup);
            }       
        },
        toggleTemplate(template) {
            if( template != this.preferredTemplate){
                this.preferredTemplate = template;
                this.updateCookie("preferred_theme", template);
                window.location.reload();
            }
        },
        updateCookie(name, value) {
            document.cookie = name + "=" + value +"; path=/";
        },
        getCookie(name) {
            const cookies = document.cookie.split(';');
            let cookieValue = null;
            for (let i = 0; i < cookies.length; i++) {
                const cookie = cookies[i].trim();
                if (cookie.startsWith(name + '=')) {
                    cookieValue = cookie.substring(name.length + 1, cookie.length);
                    break;
                }
            }
            return cookieValue;
        },
        setTheme(title, store = false) {
            this.theme = title;
            document.body.setAttribute("theme", title);
            if (store) {
                localStorage.setItem("theme", title);
                this.isSystemTheme = false;
                if(this.$root.page == 'index') {
                    this.$root.reloadCharts(); //重新载入echarts图表
                }
            }
        },
        setSystemTheme() {
            localStorage.removeItem("theme");
            this.initTheme();
            this.isSystemTheme = true;
        },
        initTheme() {
            const storeTheme = localStorage.getItem("theme");
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
        logOut(id) {
            $.ajax({
                type: 'POST',
                url: '/api/logout',
                data: JSON.stringify({ id: id }),
                contentType: 'application/json',
                success: function (resp) {
                    if (resp.code == 200) {
                        window.location.reload();
                    } else {
                        alert('注销失败(Error ' + resp.code + '): ' + resp.message);
                    }
                },
                error: function (err) {
                    alert('网络错误: ' + err.responseText);
                }
            });
        },
        goTop() {
            $('html, body').animate({ scrollTop: 0 }, 400);
            return false;
        },
        handleScroll() {
            this.showGoTop = window.scrollY >= 100;
        },
        groupingData(data, field) {
            let map = new Map();
            let dest = [];

            data.forEach(item => {
                if (!map.has(item[field])) {
                    dest.push({
                        [field]: item[field],
                        data: [item]
                    });
                    map.set(item[field], item);
                } else {
                    dest.find(dItem => dItem[field] === item[field]).data.push(item);
                }
            });
            return dest;
        },
        checkIsMobile() { // 检测设备类型,页面宽度小于768px认为是移动设备
            return window.innerWidth <= 768;
        }
    }
}