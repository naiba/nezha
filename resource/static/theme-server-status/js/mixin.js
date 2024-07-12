const mixinsVue = {
    data: {
        cache: [],
        theme: "light",
        isSystemTheme: false,
        showGroup: false,
        showGoTop: false,
        preferredTemplate: null,
        isMobile: false,
        staticUrl: '/static/theme-server-status',
        adaptedTemplates: [
            { key: 'default', name: 'Default', icon: 'th large' },
            { key: 'angel-kanade', name: 'AngelKanade', icon: 'square' },
            { key: 'server-status', name: 'ServerStatus', icon: 'list' }
        ]
    },
    created() {
        this.isMobile = this.checkIsMobile();
        this.theme = this.initTheme();
        this.showGroup = this.initShowGroup();
        this.preferredTemplate = this.getCookie('preferred_theme') ? this.getCookie('preferred_theme') : this.$root.defaultTemplate;
        window.addEventListener('scroll', this.handleScroll);
    },
    destroyed() {
        window.removeEventListener('scroll', this.handleScroll);
    },
    methods: {
        toggleTemplate(template) {
            if( template != this.preferredTemplate){
                this.preferredTemplate = template;
                this.updateCookie("preferred_theme", template);
                window.location.reload();
            }
        },
        initTheme() {
            const storedTheme = localStorage.getItem("theme");
            const theme = (storedTheme === 'dark' || storedTheme === 'light') ? storedTheme : (window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light');
            this.setTheme(theme);
            return theme;
        },
        setTheme(theme) {
            document.body.setAttribute("theme", theme);
            this.theme = theme;
            localStorage.setItem("theme", theme);
            // 重新赋值全局调色
            this.colors = this.theme == "dark" ? this.colorsDark : this.colorsLight;
            
            if(this.$root.page == 'index') {
                this.reloadCharts(); // 重新载入echarts图表
            }
        },
        initShowGroup() {
            const storedShowGroup = localStorage.getItem("showGroup");
            const showGroup = storedShowGroup !== null ? JSON.parse(storedShowGroup) : false;
            if (storedShowGroup === null) {
                localStorage.setItem("showGroup", showGroup);
            }
            return showGroup;
        },
        toggleShowGroup() {
            this.showGroup = !this.showGroup;
            localStorage.setItem("showGroup", this.showGroup);
            if (this.$root.page == 'service') {
                this.$root.initTooltip();
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