// 添加权限资源
function addPerm(userId) {
    if(!userId){
        userId = 0
    }
    var url = "/system/perm/role/add"
    var result = get({PermId:userId}, url)
    $("#add_groups_html").html(result)
    $("#add_post_html").modal("toggle")
}


/**
 * 删除权限管理弹出框
 * 2018-01-20 18:09
 */
function deletePermSwal(id) {
    Swal("删除权限管理", "warning", "确认操作", "不操作", "成功", "失败", " deletePerm("+id+")", "loadPermData()");
}


/**
 * 加载数据
 * @param key
 */
function loadPermData(key) {
    if (!key) {
        key = $("#search_user_id").val();
    } else {
        if (key.length < 4) {
            return
        }
    }
    if(!key){
        key = "";
    }

    $("#user-data-table").dataTable({
        "filter": false,//去掉搜索框
        "ordering": false, // 是否允许排序
        "paginationType": "full_numbers", // 页码类型
        "destroy": true,
        "processing": true,
        "bPaginate": true, //是否显示（应用）分页器
        "serverSide": true,
        "bLengthChange": false,
        "bInfo": true, //是否显示页脚信息，DataTables插件左下角显示记录数
        "scrollX": true, // 是否允许左右滑动
        "displayLength": 10, // 默认长度
        "ajax": { // 请求地址
            "url": "/api/perm/role?t=" + new Date().getTime() + "&search=" + key ,
            "type": 'get'
        },
        "columns": [ // 数据映射
            {"data": "RealName"},
            {"data": "Description"},
            {"data": "CreateTime"},
            {"data": "LastModifyTime"},
            {
                "sWidth": "150px", "data": "PermId", "mRender": function (data) {
                    return '<button type="button" title="更新" onclick="addPerm(' + data + ')" class="btn btn-xs rb-btn-oper"><i class="fa fa-pencil"></i></button>&nbsp;' +
                        '<button type="button"  title="删除" onClick="deletePermSwal(' + data + ')" class="delete-groups btn btn-xs rb-btn-oper"><i class="fa fa-trash-o"></i></button>';
            }
            },
        ],
        "fnRowCallback": function (row, data) { // 每行创建完毕的回调
            $(row).data('recordId', data.recordId);
        }
    });
}



/**
 * 删除权限管理方法
 * @param id
 * @return {*}
 */
function deletePerm(id) {
    var url = "/api/perm/role/"+id
    var result = del({}, url)
    result = JSON.stringify(result)
    return result
}



/**
 * 保存权限管理
 */
function savePerm(userId) {
    if(!userId){
        userId = 0
    }
    var data = get_form_data();
    data["PermId"] = parseInt(userId);
    if(!checkValue(data,"PermName")){
        return
    }
    var url = "/api/perm/role";
    var result = post(data, url);
    result = JSON.stringify(result);
    if (result.indexOf("保存成功") != -1){
        $("#add_post_html").modal("toggle");
        success(result);
        loadPermData();
    }else{
        faild(result);
    }
}
