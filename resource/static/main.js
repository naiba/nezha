const confirmBtn = $('.mini.confirm.modal .positive.button')

function showConfirm(title, content, callFn, extData) {
    const modal = $('.mini.confirm.modal')
    modal.children('.header').text(title)
    modal.children('.content').text(content)
    if (confirmBtn.hasClass('loading')) {
        return false
    }
    modal.modal({
        closable: true,
        onApprove: function () {
            confirmBtn.toggleClass('loading')
            callFn(extData)
            return false
        }
    }).modal('show')
}

function showFormModal(modelSelector, formID, URL, getData) {
    $(modelSelector).modal({
        closable: true,
        onApprove: function () {
            let success = false
            const btn = $(modelSelector + ' .positive.button')
            const form = $(modelSelector + ' form')
            if (btn.hasClass('loading')) {
                return success
            }
            form.children('.message').remove()
            btn.toggleClass('loading')
            const data = getData ? getData() : $(formID).serializeArray().reduce(function (obj, item) {
                obj[item.name] = (item.name.endsWith('_id') || item.name === 'id') ? parseInt(item.value) : item.value;
                return obj;
            }, {});
            $.post(URL, JSON.stringify(data)).done(function (resp) {
                if (resp.code == 200) {
                    if (resp.message) {
                        alert(resp.message)
                    }
                    window.location.reload()
                } else {
                    form.append(`<div class="ui negative message"><div class="header">操作失败</div><p>` + resp.message + `</p></div>`)
                }
            }).fail(function (err) {
                form.append(`<div class="ui negative message"><div class="header">网络错误</div><p>` + err.responseText + `</p></div>`)
            }).always(function () {
                btn.toggleClass('loading')
            });
            return success
        }
    }).modal('show')
}

function addServer() {
    showFormModal('.server.modal', '#serverForm', '/api/server')
}

function logout(id) {
    $.post('/api/logout', JSON.stringify({ id: id })).done(function (resp) {
        if (resp.code == 200) {
            $.suiAlert({
                title: '注销成功',
                type: 'success',
                time: '3',
                position: 'top-center',
            });
            window.location.reload()
        } else {
            $.suiAlert({
                title: '注销失败',
                description: resp.code + '：' + resp.message,
                type: 'error',
                time: '3',
                position: 'top-center',
            });
        }
    }).fail(function (err) {
        alert('网络错误：' + err.responseText)
    })
}
