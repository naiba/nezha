const mixinsVue = {
    delimiters: ['@#', '#@'],
    data: {
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
        this.preferredTemplate = this.getCookie('preferred_theme') ? this.getCookie('preferred_theme') : this.$root.defaultTemplate;
    },
    mounted() {
        this.initDropdown();
    },
    methods: {
        initDropdown() {
            if(this.isMobile) $('.ui.dropdown').dropdown({
                action: 'hide',
                on: 'click',
                duration: 100,
                direction: 'direction'
            });
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
        checkIsMobile() { // 检测设备类型,页面宽度小于768px认为是移动设备
            return window.innerWidth <= 768;
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
        }
    }
}