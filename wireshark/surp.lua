local surp_proto = Proto("surp", "SURP Protocol")

-- Define protocol fields
local f_magic         = ProtoField.string("surp.magic", "Magic", base.ASCII)
local f_msg_type      = ProtoField.uint8("surp.msg_type", "Message Type", base.HEX, {
    [0x01] = "Advertise",
    [0x02] = "Update",
    [0x03] = "Set"
})
local f_seq           = ProtoField.uint16("surp.seq", "Sequence Number", base.DEC)
local f_group_len     = ProtoField.uint8("surp.group_len", "Group Name Length", base.DEC)
local f_group         = ProtoField.string("surp.group", "Group Name", base.ASCII)
local f_port          = ProtoField.uint16("surp.port", "Port", base.DEC)
local f_reg_count     = ProtoField.uint16("surp.reg_count", "Register Count", base.DEC)
local f_reg_name_len  = ProtoField.uint8("surp.reg_name_len", "Register Name Length", base.DEC)
local f_reg_name      = ProtoField.string("surp.reg_name", "Register Name", base.ASCII)
local f_val_len       = ProtoField.uint16("surp.val_len", "Value Length", base.DEC)
local f_val           = ProtoField.bytes("surp.val", "Value")
local f_meta_count    = ProtoField.uint8("surp.meta_count", "Metadata Count", base.DEC)
local f_meta_key_len  = ProtoField.uint8("surp.meta_key_len", "Metadata Key Length", base.DEC)
local f_meta_key      = ProtoField.string("surp.meta_key", "Metadata Key", base.ASCII)
local f_meta_val_len  = ProtoField.uint8("surp.meta_val_len", "Metadata Value Length", base.DEC)
local f_meta_val      = ProtoField.string("surp.meta_val", "Metadata Value", base.ASCII)

local f_reg_name2_len = ProtoField.uint16("surp.reg_name2_len", "Register Name Length", base.DEC)
local f_reg_name2     = ProtoField.string("surp.reg_name2", "Register Name", base.ASCII)

surp_proto.fields = {
    f_magic, f_msg_type, f_seq, f_group_len, f_group, f_port, f_reg_count,
    f_reg_name_len, f_reg_name, f_val_len, f_val, f_meta_count,
    f_meta_key_len, f_meta_key, f_meta_val_len, f_meta_val,
    f_reg_name2_len, f_reg_name2
}

-- Main dissector function
function surp_proto.dissector(tvb, pinfo, tree)
    if tvb:len() < 5 then return end
    local offset = 0

    -- Check magic number
    local magic = tvb(offset,4):string()
    if magic ~= "SURP" then return end

    pinfo.cols.protocol = "SURP"
    local subtree = tree:add(surp_proto, tvb(),"SURP Protocol Data")
    subtree:add(f_magic, tvb(offset,4))
    offset = offset + 4

    local msg_type = tvb(offset,1):uint()
    subtree:add(f_msg_type, tvb(offset,1))
    offset = offset + 1

    local info_str = ""
    if msg_type == 0x01 then

        -- Advertise Message
        if tvb:len() < offset + 2 then return end
        subtree:add(f_seq, tvb(offset,2))
        offset = offset + 2

        if tvb:len() < offset + 1 then return end
        local grp_len = tvb(offset,1):uint()
        subtree:add(f_group_len, tvb(offset,1))
        offset = offset + 1

        if tvb:len() < offset + grp_len then return end
        local group_name = tvb(offset,grp_len):string()
        subtree:add(f_group, tvb(offset,grp_len))
        offset = offset + grp_len

        info_str = group_name .. " advertise "

        if tvb:len() < offset + 2 then return end
        subtree:add(f_port, tvb(offset,2))
        offset = offset + 2

        if tvb:len() < offset + 1 then return end
        local reg_count = tvb(offset,1):uint()
        subtree:add(f_reg_count, tvb(offset,1))
        offset = offset + 1

        -- Process each register
        for i = 1, reg_count do

            if tvb:len() < offset + 1 then return end
            local reg_name_len = tvb(offset,1):uint()
            reg_offset = offset
            offset = offset + 1

            if tvb:len() < offset + reg_name_len then return end
            local reg_name = tvb(offset,reg_name_len):string()
            offset = offset + reg_name_len

            local reg_tree = subtree:add(surp_proto, tvb(reg_offset), "Register "..reg_name)
            reg_tree:add(f_reg_name_len, tvb(reg_offset,1))
            reg_tree:add(f_reg_name, tvb(reg_offset + 1,reg_name_len))

            local reg_info = reg_name .. "="

            if tvb:len() < offset + 2 then return end
            local val_len = tvb(offset,2):int()
            reg_tree:add(f_val_len, tvb(offset,2))
            offset = offset + 2

            local val_str = ""
            if val_len == -1 then
                val_str = "(undefined)"
            elseif val_len >= 0 then
                if tvb:len() < offset + val_len then return end
                local val_hex = ""
                for i = 0, val_len - 1 do
                    val_hex = val_hex .. string.format("%02X", tvb(offset + i, 1):uint())
                end
                val_str = val_hex
                reg_tree:add(f_val, tvb(offset, val_len))
                offset = offset + val_len
            end
            reg_info = reg_info .. val_str

            if tvb:len() < offset + 1 then return end
            local meta_count = tvb(offset,1):uint()
            reg_tree:add(f_meta_count, tvb(offset,1))
            offset = offset + 1

            local meta_str = ""
            for j = 1, meta_count do
                if tvb:len() < offset + 1 then return end
                local key_len = tvb(offset,1):uint()
                local meta_key_offset = offset
                offset = offset + 1

                if tvb:len() < offset + key_len then return end
                local meta_key = tvb(offset, key_len):string()
                offset = offset + key_len

                if tvb:len() < offset + 1 then return end
                local val_key_len = tvb(offset,1):uint()
                local meta_val_offset = offset
                offset = offset + 1

                if tvb:len() < offset + val_key_len then return end
                local meta_val = tvb(offset, val_key_len):string()
                offset = offset + val_key_len

                meta_str = meta_str .. " "..meta_key .. ":" .. meta_val

                local meta_tree = reg_tree:add(surp_proto, tvb(meta_key_offset), "Metadata " .. meta_key..":"..meta_val)
                meta_tree:add(f_meta_key_len, tvb(meta_key_offset,1))
                meta_tree:add(f_meta_key, tvb(meta_key_offset+1, key_len))
                meta_tree:add(f_meta_val_len, tvb(meta_val_offset,1))
                meta_tree:add(f_meta_val, tvb(meta_val_offset+1, val_key_len))
            end
            reg_info = reg_info .. meta_str
            if i > 1 then
                info_str = info_str .. ", "
            end
            info_str = info_str .. reg_info
        end

    elseif msg_type == 0x02 or msg_type == 0x03 then

        -- Update or Set Message
        if tvb:len() < offset + 2 then return end
        subtree:add(f_seq, tvb(offset,2))
        offset = offset + 2

        if tvb:len() < offset + 1 then return end
        local grp_len = tvb(offset,1):uint()
        subtree:add(f_group_len, tvb(offset,1))
        offset = offset + 1

        if tvb:len() < offset + grp_len then return end
        local group_name = tvb(offset,grp_len):string()
        subtree:add(f_group, tvb(offset,grp_len))
        offset = offset + grp_len

        info_str = group_name .. (msg_type == 0x02 and " update " or " set ")

        if tvb:len() < offset + 1 then return end
        local reg_count = tvb(offset,1):uint()
        subtree:add(f_reg_count, tvb(offset,1))
        offset = offset + 1

        for i = 1, reg_count do

            if tvb:len() < offset + 1 then return end
            local reg_name_len = tvb(offset,1):uint()
            local reg_offset = offset
            offset = offset + 1

            if tvb:len() < offset + reg_name_len then return end
            local reg_name = tvb(offset,reg_name_len):string()
            offset = offset + reg_name_len

            local reg_tree = subtree:add(surp_proto, tvb(reg_offset), "Register "..reg_name)
            reg_tree:add(f_reg_name_len, tvb(reg_offset,1))
            reg_tree:add(f_reg_name, tvb(reg_offset + 1,reg_name_len))

            if tvb:len() < offset + 2 then return end
            local val_len = tvb(offset,2):int()
            reg_tree:add(f_val_len, tvb(offset,2))
            offset = offset + 2

            local val_str = ""
            if val_len == -1 then
                val_str = "(undefined)"
            elseif val_len >= 0 then
                if tvb:len() < offset + val_len then return end
                local val_hex = ""
                for i = 0, val_len - 1 do
                    val_hex = val_hex .. string.format("%02X", tvb(offset + i, 1):uint())
                end
                val_str = val_hex
                reg_tree:add(f_val, tvb(offset, val_len))
                offset = offset + val_len
            end

            if i > 1 then
                info_str = info_str .. ", "
            end
            info_str = info_str .. reg_name .. "=" .. val_str
        end

    elseif msg_type == 0x04 then
        -- Join Message

        if tvb:len() < offset + 2 then return end
        subtree:add(f_seq, tvb(offset,2))
        offset = offset + 2

        if tvb:len() < offset + 1 then return end
        local grp_len = tvb(offset,1):uint()
        subtree:add(f_group_len, tvb(offset,1))
        offset = offset + 1

        if tvb:len() < offset + grp_len then return end
        local group_name = tvb(offset,grp_len):string()
        subtree:add(f_group, tvb(offset,grp_len))
        offset = offset + grp_len

        info_str = group_name .. " join "

    else
        subtree:add_expert_info(PI_MALFORMED, PI_ERROR, "Unknown SURP message type")
        info_str = info_str .. "Unknown"
    end

    pinfo.cols.info = info_str
end

-- Heuristic dissector function
function check_heuristic(tvb, pinfo, tree)
    if tvb:len() < 4 then return false end
    if tvb(0,4):string() == "SURP" then
        surp_proto.dissector(tvb, pinfo, tree)
        return true
    end
    return false
end

surp_proto:register_heuristic("udp", check_heuristic)
