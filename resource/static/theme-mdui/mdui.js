function showConfirm(title, content, callFn, extData) {
  mdui.confirm(content, title,
    function(){
      callFn(extData);
    });
}

function logout(id) {
  $.post("/api/logout", JSON.stringify({ id: id }))
    .done(function (resp) {
      if (resp.code == 200) {
        mdui.snackbar({
          message: '注销成功',
          timeout: 2000,
          position: 'top',
        });
        window.location.reload();
      } else {
        mdui.snackbar({
          message: '注销失败(Error ' + resp.code + '): ' + resp.message,
          timeout: 5000,
          position: 'top',
        });
      }
    })
    .fail(function (err) {
      mdui.snackbar({
        message: '网络错误: ' + err.responseText,
        timeout: 5000,
        position: 'top',
      });
    });
}
